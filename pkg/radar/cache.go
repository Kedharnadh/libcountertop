package radar

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	memoryMost = 16 << 20
	expiring   = "expiring"
)

type cached struct {
	key     string
	body    []byte
	expires time.Time
}

type cache struct {
	dir string

	mu    sync.Mutex
	order *list.List
	items map[string]*list.Element
	bytes int
}

func newCache(dir string) *cache {
	c := &cache{dir: dir, order: list.New(), items: map[string]*list.Element{}}
	c.prune()
	return c
}

func (c *cache) get(url string, keep bool) ([]byte, bool) {
	if keep {
		if c.dir == "" {
			return nil, false
		}
		b, err := os.ReadFile(c.path(url, keep))
		return b, err == nil
	}
	if b, ok := c.held(url); ok {
		return b, true
	}
	if c.dir == "" {
		return nil, false
	}
	path := c.path(url, keep)
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if time.Now().After(info.ModTime()) {
		_ = os.Remove(path)
		return nil, false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	c.hold(url, b, info.ModTime())
	return b, true
}

func (c *cache) put(url string, body []byte, keep bool, until time.Time) {
	if !keep {
		c.hold(url, body, until)
	}
	if c.dir == "" {
		return
	}
	path := c.path(url, keep)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return
	}
	tmp := path + ".part"
	if os.WriteFile(tmp, body, 0o644) != nil {
		return
	}
	if !keep && os.Chtimes(tmp, until, until) != nil {
		_ = os.Remove(tmp)
		return
	}
	_ = os.Rename(tmp, path)
}

func (c *cache) held(url string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.items[url]
	if !ok {
		return nil, false
	}
	item := e.Value.(*cached)
	if time.Now().After(item.expires) {
		c.drop(e)
		return nil, false
	}
	c.order.MoveToFront(e)
	return item.body, true
}

func (c *cache) hold(url string, body []byte, until time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e, ok := c.items[url]; ok {
		c.drop(e)
	}
	c.items[url] = c.order.PushFront(&cached{key: url, body: body, expires: until})
	c.bytes += len(body)
	for c.bytes > memoryMost && c.order.Len() > 0 {
		c.drop(c.order.Back())
	}
}

func (c *cache) prune() {
	if c.dir == "" {
		return
	}
	now := time.Now()
	_ = filepath.WalkDir(
		filepath.Join(c.dir, expiring),
		func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if info, err := d.Info(); err == nil && now.After(info.ModTime()) {
				_ = os.Remove(path)
			}
			return nil
		},
	)
}

func (c *cache) drop(e *list.Element) {
	item := e.Value.(*cached)
	c.order.Remove(e)
	delete(c.items, item.key)
	c.bytes -= len(item.body)
}

func (c *cache) path(url string, keep bool) string {
	sum := sha256.Sum256([]byte(url))
	name := hex.EncodeToString(sum[:])
	if keep {
		return filepath.Join(c.dir, name[:2], name+".tile")
	}
	return filepath.Join(c.dir, expiring, name[:2], name+".tile")
}

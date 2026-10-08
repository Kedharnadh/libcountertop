package sendspin

import (
	"bytes"
	"fmt"
	"io"
	"sync"

	"github.com/mewkiz/flac"
)

type chunkFeed struct {
	mu     sync.Mutex
	ready  *sync.Cond
	hungry *sync.Cond

	pending []byte
	wants   bool
	closed  bool
}

func newChunkFeed() *chunkFeed {
	f := &chunkFeed{}
	f.ready = sync.NewCond(&f.mu)
	f.hungry = sync.NewCond(&f.mu)
	return f
}

func (f *chunkFeed) Read(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for len(f.pending) == 0 && !f.closed {
		f.wants = true
		f.hungry.Broadcast()
		f.ready.Wait()
	}
	if len(f.pending) == 0 {
		return 0, io.EOF
	}

	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}

func (f *chunkFeed) feed(chunk []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.closed {
		return io.ErrClosedPipe
	}

	f.pending = chunk
	f.wants = false
	f.ready.Broadcast()

	for !f.wants && !f.closed {
		f.hungry.Wait()
	}
	if f.closed {
		return io.ErrClosedPipe
	}
	return nil
}

func (f *chunkFeed) stop() {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.closed = true
	f.pending = nil
	f.ready.Broadcast()
	f.hungry.Broadcast()
}

type flacDecoder struct {
	feed *chunkFeed
	done chan struct{}

	mu    sync.Mutex
	queue [][]int16
	err   error
}

func newFLACDecoder(header []byte) (decoder, error) {
	if len(header) == 0 {
		return nil, fmt.Errorf("sendspin: flac needs the codec header stream/start carries")
	}

	d := &flacDecoder{
		feed: newChunkFeed(),
		done: make(chan struct{}),
	}

	go d.parse(header)
	return d, nil
}

func (d *flacDecoder) parse(header []byte) {
	defer close(d.done)
	defer d.feed.stop()

	stream, err := flac.New(io.MultiReader(bytes.NewReader(header), d.feed))
	if err != nil {
		d.failed(fmt.Errorf("sendspin: flac header: %w", err))
		return
	}

	channels := int(stream.Info.NChannels)
	shift := int(stream.Info.BitsPerSample) - Bits

	for {
		frame, err := stream.ParseNext()
		if err != nil {
			return
		}

		pcm := make([]int16, int(frame.BlockSize)*channels)
		for i := range int(frame.BlockSize) {
			for c := range channels {
				s := frame.Subframes[c].Samples[i]
				if shift > 0 {
					s >>= shift
				}
				pcm[i*channels+c] = int16(s)
			}
		}
		d.push(pcm)
	}
}

func (d *flacDecoder) push(pcm []int16) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.queue = append(d.queue, pcm)
}

func (d *flacDecoder) pop() []int16 {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.queue) == 0 {
		return nil
	}
	pcm := d.queue[0]
	d.queue = d.queue[1:]
	return pcm
}

func (d *flacDecoder) failed(err error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.err = err
}

func (d *flacDecoder) failure() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.err
}

func (d *flacDecoder) decode(chunk []byte) ([]int16, error) {
	if err := d.failure(); err != nil {
		return nil, err
	}

	if err := d.feed.feed(chunk); err != nil {
		if perr := d.failure(); perr != nil {
			return nil, perr
		}
		return nil, io.EOF
	}
	return d.pop(), nil
}

func (d *flacDecoder) close() error {
	d.feed.stop()
	<-d.done
	return nil
}

package releases

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

const (
	feedTimeout = 10 * time.Second
	maxFeed     = 1 << 20
)

// The releases.atom feed lists every published release, prereleases included, newest first.
func Newest(ctx context.Context, repo string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, feedTimeout)
	defer cancel()
	url := strings.TrimSuffix(repo, "/") + "/releases.atom"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("releases: fetching %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("releases: fetching %s: %s", url, resp.Status)
	}
	return Parse(io.LimitReader(resp.Body, maxFeed), Skip)
}

var Skip = []string{"dev", "boot-images"}

func Parse(r io.Reader, skip []string) (string, error) {
	var feed struct {
		Entries []struct {
			Link struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
		} `xml:"entry"`
	}
	if err := xml.NewDecoder(r).Decode(&feed); err != nil {
		return "", fmt.Errorf("releases: reading the feed: %w", err)
	}
	for _, e := range feed.Entries {
		_, tag, ok := strings.Cut(e.Link.Href, "/releases/tag/")
		if !ok || tag == "" || slices.Contains(skip, tag) {
			continue
		}
		return tag, nil
	}
	return "", errors.New("releases: the feed lists no release")
}

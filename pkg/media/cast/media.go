package cast

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// CASTV2 carries times as floating point seconds.

const (
	StateIdle      = "IDLE"
	StateBuffering = "BUFFERING"
	StatePlaying   = "PLAYING"
	StatePaused    = "PAUSED"
)

const (
	IdleFinished    = "FINISHED"
	IdleCancelled   = "CANCELLED"
	IdleInterrupted = "INTERRUPTED"
	IdleError       = "ERROR"
)

const (
	StreamBuffered = "BUFFERED"
	StreamLive     = "LIVE"
	StreamNone     = "NONE"
)

const (
	MetadataGeneric = 0
	MetadataMovie   = 1
	MetadataTVShow  = 2
	MetadataMusic   = 3
	MetadataPhoto   = 4
)

const (
	CommandPause        = 1 << 0
	CommandSeek         = 1 << 1
	CommandStreamVolume = 1 << 2
	CommandStreamMute   = 1 << 3
	CommandQueueNext    = 1 << 6
	CommandQueuePrev    = 1 << 7
)

const Speaker = CommandPause | CommandSeek | CommandStreamVolume | CommandStreamMute

type Image struct {
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

type Metadata struct {
	MetadataType int     `json:"metadataType"`
	Title        string  `json:"title,omitempty"`
	Subtitle     string  `json:"subtitle,omitempty"`
	Artist       string  `json:"artist,omitempty"`
	AlbumName    string  `json:"albumName,omitempty"`
	AlbumArtist  string  `json:"albumArtist,omitempty"`
	Composer     string  `json:"composer,omitempty"`
	TrackNumber  int     `json:"trackNumber,omitempty"`
	DiscNumber   int     `json:"discNumber,omitempty"`
	Images       []Image `json:"images,omitempty"`
	ReleaseDate  string  `json:"releaseDate,omitempty"`
}

type Media struct {
	ContentID   string `json:"contentId"`
	ContentURL  string `json:"contentUrl,omitempty"`
	StreamType  string `json:"streamType"`
	ContentType string `json:"contentType"`
	Live        bool   `json:"isLiveMedia,omitempty"`

	Metadata Metadata `json:"metadata,omitempty"`

	Duration float64 `json:"duration,omitempty"`

	CustomData json.RawMessage `json:"customData,omitempty"`
}

func (m Media) URL() string {
	if m.ContentURL != "" {
		return m.ContentURL
	}
	return m.ContentID
}

func (m Media) Streaming() bool { return m.Live || m.StreamType == StreamLive }

func (m Media) Category() string {
	switch {
	case strings.HasPrefix(m.ContentType, "audio/"):
		return "AUDIO"
	case strings.HasPrefix(m.ContentType, "video/"):
		return "VIDEO"
	}
	return ""
}

func (m Media) Length() time.Duration { return seconds(m.Duration) }

func (m Media) Title() string {
	if m.Metadata.Title != "" {
		return m.Metadata.Title
	}
	return m.ContentID
}

func seconds(v float64) time.Duration { return time.Duration(v * float64(time.Second)) }

func asSeconds(d time.Duration) float64 { return d.Seconds() }

type MediaStatus struct {
	MediaSessionID int `json:"mediaSessionId"`

	PlaybackRate float64 `json:"playbackRate"`
	PlayerState  string  `json:"playerState"`

	CurrentTime float64 `json:"currentTime"`

	SupportedMediaCommands int             `json:"supportedMediaCommands"`
	Volume                 Volume          `json:"volume"`
	MediaCategory          string          `json:"mediaCategory,omitempty"`
	CustomData             json.RawMessage `json:"customData,omitempty"`

	Media *Media `json:"media,omitempty"`

	IdleReason string `json:"idleReason,omitempty"`

	CurrentItemID int `json:"currentItemId,omitempty"`
}

func (s MediaStatus) Elapsed() time.Duration { return seconds(s.CurrentTime) }

// Senders treat a status that is not an array as malformed.
type mediaStatusEnvelope struct {
	Header
	Status []MediaStatus `json:"status"`
}

type mediaBroadcastEnvelope struct {
	Type   string        `json:"type"`
	Status []MediaStatus `json:"status"`
}

func MediaBroadcastPayload(s ...MediaStatus) string {
	if s == nil {
		s = []MediaStatus{}
	}
	return payload(mediaBroadcastEnvelope{Type: TypeMediaStatus, Status: s})
}

func MediaStatusPayload(request int, s ...MediaStatus) string {
	// Senders read null as malformed and [] as nothing playing.
	if s == nil {
		s = []MediaStatus{}
	}
	return payload(mediaStatusEnvelope{
		Header: Header{Type: TypeMediaStatus, RequestID: request},
		Status: s,
	})
}

type LoadRequest struct {
	Header

	Media Media `json:"media"`

	// Absent means the protocol default of true.
	Autoplay *bool `json:"autoplay,omitempty"`

	CurrentTime float64 `json:"currentTime,omitempty"`

	CustomData json.RawMessage `json:"customData,omitempty"`
}

func (r LoadRequest) Starts() bool { return r.Autoplay == nil || *r.Autoplay }

func ParseLoad(p string) (LoadRequest, error) {
	var r LoadRequest
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r, fmt.Errorf("cast: reading a load: %w", err)
	}
	if r.Media.ContentID == "" && r.Media.ContentURL == "" {
		return r, fmt.Errorf("cast: a load naming nothing to play")
	}
	return r, nil
}

type MediaRequest struct {
	Header

	MediaSessionID int `json:"mediaSessionId"`

	CurrentTime float64 `json:"currentTime,omitempty"`

	ResumeState string `json:"resumeState,omitempty"`

	Jump int `json:"jump,omitempty"`
}

const (
	ResumePlay  = "PLAYBACK_START"
	ResumePause = "PLAYBACK_PAUSE"
)

func ParseMediaRequest(p string) (MediaRequest, error) {
	var r MediaRequest
	if err := json.Unmarshal([]byte(p), &r); err != nil {
		return r, fmt.Errorf("cast: reading a media command: %w", err)
	}
	return r, nil
}

func (r MediaRequest) Seek() time.Duration { return seconds(r.CurrentTime) }

func LoadFailed(request int) string {
	return payload(Header{Type: TypeLoadFailed, RequestID: request})
}

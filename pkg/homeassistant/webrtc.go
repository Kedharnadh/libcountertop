package homeassistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	WebRTCProtocol = 2

	webrtcOpen  = "echolocal.camera_webrtc"
	webrtcClose = "echolocal.camera_webrtc_close"
)

var ErrNoWebRTC = errors.New("homeassistant: no connected Home Assistant relays camera WebRTC")

type WebRTC struct {
	Entity  string
	Session string
	Answer  string
	via     sender
}

func (h *Controller) OpenWebRTC(ctx context.Context, entity, offer string) (WebRTC, error) {
	conn, ok := h.speaker(WebRTCProtocol)
	if !ok {
		return WebRTC{}, ErrNoWebRTC
	}
	raw, err := h.send(
		ctx,
		[]sender{conn},
		request(webrtcOpen, map[string]string{"entity_id": entity, "offer": offer}, nil, pingReply),
	)
	if err != nil {
		return WebRTC{}, err
	}
	var out struct {
		Response json.RawMessage `json:"response"`
	}
	var answer struct {
		Session string `json:"session"`
		Answer  string `json:"answer"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return WebRTC{}, fmt.Errorf("homeassistant: the webrtc response: %w", err)
	}
	if err := json.Unmarshal(unwrap(out.Response), &answer); err != nil {
		return WebRTC{}, fmt.Errorf("homeassistant: the webrtc answer: %w", err)
	}
	if answer.Answer == "" {
		return WebRTC{}, errors.New("homeassistant: the webrtc answer is empty")
	}
	return WebRTC{Entity: entity, Session: answer.Session, Answer: answer.Answer, via: conn}, nil
}

func (h *Controller) CloseWebRTC(ctx context.Context, w WebRTC) error {
	if w.via == nil || w.Session == "" {
		return nil
	}
	_, err := h.send(
		ctx,
		[]sender{w.via},
		request(
			webrtcClose,
			map[string]string{"entity_id": w.Entity, "session": w.Session},
			nil,
			pingReply,
		),
	)
	return err
}

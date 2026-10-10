package cast

import (
	"encoding/json"
	"fmt"
)

const NSDiscovery = "urn:x-cast:com.google.cast.receiver.discovery"

const (
	TypeGetDeviceInfo = "GET_DEVICE_INFO"
	TypeDeviceInfo    = "DEVICE_INFO"
)

type deviceInfo struct {
	Type                 string `json:"type"`
	RequestID            int    `json:"requestId"`
	DeviceID             string `json:"deviceId"`
	FriendlyName         string `json:"friendlyName"`
	DeviceModel          string `json:"deviceModel"`
	DeviceCapabilities   int    `json:"deviceCapabilities"`
	DeviceIconURL        string `json:"deviceIconUrl"`
	ControlNotifications int    `json:"controlNotifications"`
	ReceiverMetricsID    string `json:"receiverMetricsId,omitempty"`
	WifiProximityID      string `json:"wifiProximityId"`
}

func (r *Receiver) discovery(m Message) ([]Message, error) {
	h, err := Kind(m.Payload)
	if err != nil {
		return nil, err
	}
	if h.Type != TypeGetDeviceInfo || r.Identity == nil {
		if r.Unspoken != nil {
			r.Unspoken(m.Namespace, h.Type)
		}
		return nil, nil
	}

	d := r.Identity()
	body, err := json.Marshal(deviceInfo{
		Type:                 TypeDeviceInfo,
		RequestID:            h.RequestID,
		DeviceID:             d.ID,
		FriendlyName:         d.Name,
		DeviceModel:          d.Model,
		DeviceCapabilities:   d.Profile.Info,
		DeviceIconURL:        IconPath,
		ControlNotifications: 1,
		WifiProximityID:      d.Proximity(),
	})
	if err != nil {
		return nil, fmt.Errorf("cast: device info: %w", err)
	}
	return []Message{reply(m, NSDiscovery, string(body))}, nil
}

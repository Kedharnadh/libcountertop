package cast

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const ServiceType = "_googlecast._tcp"

const Port = 8009

const (
	CapabilityVideoOut  = 1 << 0
	CapabilityVideoIn   = 1 << 1
	CapabilityAudioOut  = 1 << 2
	CapabilityAudioIn   = 1 << 3
	CapabilityDevMode   = 1 << 4
	CapabilityMultizone = 1 << 5
)

// The ca a stock Lenovo Smart Display 10 and a Google Nest Hub both advertise.
const SmartDisplay = 231941

const Version = "05"

type Device struct {
	ID string

	Name string

	Model string

	Capabilities int

	Running bool

	Status string
}

func Identify(mac, name, model string) Device {
	clean := strings.ReplaceAll(strings.ToLower(mac), ":", "")

	return Device{
		ID:           derive("cast-id", clean),
		Name:         name,
		Model:        model,
		Capabilities: SmartDisplay,
	}
}

func derive(purpose, mac string) string {
	sum := sha256.Sum256([]byte(purpose + ":" + mac))
	return hex.EncodeToString(sum[:16])
}

func (d Device) Records() []string {
	status := "0"
	if d.Running {
		status = "1"
	}

	return []string{
		"id=" + d.ID,
		"cd=" + d.CloudID(),
		"rm=",
		"ve=" + Version,
		"md=" + d.Model,
		"ic=" + IconPath,
		"fn=" + d.Name,
		"ca=" + fmt.Sprint(d.Capabilities),
		"st=" + status,
		"bs=" + d.Proximity(),
		"nf=1",
		"rs=" + d.Status,
	}
}

// The cd record and eureka_info cloud_device_id, uppercase as a stock device has it.
func (d Device) CloudID() string { return strings.ToUpper(derive("cast-cloud", d.ID)) }

// The bs record and DEVICE_INFO wifiProximityId.
func (d Device) Proximity() string { return strings.ToUpper(derive("cast-bootstrap", d.ID))[:12] }

func (d Device) Instance() string { return d.ID }

func (d Device) Valid() error {
	if len(d.ID) != 32 {
		return fmt.Errorf("cast: the device id is %d characters, want 32", len(d.ID))
	}
	if _, err := hex.DecodeString(d.ID); err != nil {
		return fmt.Errorf("cast: the device id is not hexadecimal")
	}
	if d.Name == "" {
		return fmt.Errorf("cast: a device with no name would be listed as nothing")
	}
	if d.Capabilities == 0 {
		return fmt.Errorf("cast: a device claiming no capabilities is not offered for anything")
	}
	return nil
}

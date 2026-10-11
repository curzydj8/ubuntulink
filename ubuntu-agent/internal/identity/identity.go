// Package identity manages the agent's persistent device identity.
package identity

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

type DeviceIdentity struct {
	DeviceID string `json:"device_id"`
}

func newDeviceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// LoadOrCreate reads dir/device.json (0600) or creates it.
func LoadOrCreate(dir string) (DeviceIdentity, error) {
	path := filepath.Join(dir, "device.json")
	if data, err := os.ReadFile(path); err == nil {
		var id DeviceIdentity
		if err := json.Unmarshal(data, &id); err == nil && id.DeviceID != "" {
			return id, nil
		}
	}
	id, err := newDeviceID()
	if err != nil {
		return DeviceIdentity{}, err
	}
	dev := DeviceIdentity{DeviceID: id}
	data, err := json.Marshal(dev)
	if err != nil {
		return DeviceIdentity{}, err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return DeviceIdentity{}, err
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return DeviceIdentity{}, err
	}
	return dev, nil
}

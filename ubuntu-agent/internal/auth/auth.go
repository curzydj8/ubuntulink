// Package auth manages paired devices and Bearer tokens.
// Tokens are 256-bit crypto-random; only their SHA-256 is stored.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type PairedDevice struct {
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	// TokenHash is the SHA-256 of the token. It cannot be reversed to
	// recover the token; the file itself is 0600.
	TokenHash string    `json:"token_hash"`
	PairedAt  time.Time `json:"paired_at"`
}

// PublicView omits anything sensitive.
type PublicView struct {
	ClientID   string    `json:"client_id"`
	ClientName string    `json:"client_name"`
	PairedAt   time.Time `json:"paired_at"`
}

type Auth struct {
	mu        sync.Mutex
	devices   map[string]PairedDevice // tokenHash -> device
	storePath string
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// New loads persisted pairings (0600 file) or starts empty.
func New(dir string) (*Auth, error) {
	a := &Auth{devices: map[string]PairedDevice{}, storePath: filepath.Join(dir, "paired.json")}
	data, err := os.ReadFile(a.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return nil, err
	}
	var list []PairedDevice
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	for _, d := range list {
		if d.TokenHash != "" {
			a.devices[d.TokenHash] = d
		}
	}
	return a, nil
}

func (a *Auth) persistLocked() error {
	list := make([]PairedDevice, 0, len(a.devices))
	for _, d := range a.devices {
		list = append(list, d)
	}
	data, err := json.Marshal(list)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(a.storePath), 0700); err != nil {
		return err
	}
	return os.WriteFile(a.storePath, data, 0600)
}

// IssueToken creates a pairing for a client and returns the raw token
// (shown to the client exactly once).
func (a *Auth) IssueToken(clientID, clientName string) (string, error) {
	if clientID == "" {
		return "", errors.New("client_id required")
	}
	token, err := newToken()
	if err != nil {
		return "", err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.devices[hashToken(token)] = PairedDevice{
		ClientID:   clientID,
		ClientName: clientName,
		TokenHash:  hashToken(token),
		PairedAt:   time.Now(),
	}
	if err := a.persistLocked(); err != nil {
		delete(a.devices, hashToken(token))
		return "", err
	}
	return token, nil
}

// Validate returns the device for a valid token.
func (a *Auth) Validate(token string) (PairedDevice, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	d, ok := a.devices[hashToken(token)]
	return d, ok
}

// Revoke removes a token immediately.
func (a *Auth) Revoke(token string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	h := hashToken(token)
	if _, ok := a.devices[h]; !ok {
		return false
	}
	delete(a.devices, h)
	_ = a.persistLocked()
	return true
}

// RevokeByClientID removes all tokens for a client (unpair a device).
func (a *Auth) RevokeByClientID(clientID string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for h, d := range a.devices {
		if d.ClientID == clientID {
			delete(a.devices, h)
			n++
		}
	}
	_ = a.persistLocked()
	return n
}

// List returns paired devices without any token material.
func (a *Auth) List() []PublicView {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := make([]PublicView, 0, len(a.devices))
	for _, d := range a.devices {
		out = append(out, PublicView{
			ClientID:   d.ClientID,
			ClientName: d.ClientName,
			PairedAt:   d.PairedAt,
		})
	}
	return out
}

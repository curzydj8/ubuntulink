// Package pairing implements short-lived pairing codes with
// attempt limiting: the user-visible step that authorizes a new device.
package pairing

import (
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrInvalidCode = errors.New("invalid pairing code")
	ErrExpired     = errors.New("pairing code expired")
	ErrLocked      = errors.New("too many attempts, try later")
)

type Pairing struct {
	mu          sync.Mutex
	code        string
	expiresAt   time.Time
	attempts    int
	maxAttempts int
	lockedUntil time.Time
	lockDur     time.Duration
	ttl         time.Duration
}

func New(ttl time.Duration, maxAttempts int, lockDur time.Duration) *Pairing {
	p := &Pairing{ttl: ttl, maxAttempts: maxAttempts, lockDur: lockDur}
	p.rotateLocked()
	return p
}

func genCode() string {
	// 6-digit numeric code, crypto-random.
	var b [3]byte
	_, _ = rand.Read(b[:])
	n := int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	return fmt.Sprintf("%06d", n%1000000)
}

func (p *Pairing) rotateLocked() {
	p.code = genCode()
	p.expiresAt = time.Now().Add(p.ttl)
	p.attempts = 0
}

// Code returns the current code, rotating it if expired.
// The code is meant to be shown to the user (log/terminal/UI).
func (p *Pairing) Code() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	if time.Now().After(p.expiresAt) {
		p.rotateLocked()
	}
	return p.code
}

// ExpiresAt reports when the current code expires.
func (p *Pairing) ExpiresAt() time.Time {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.expiresAt
}

// Verify checks input. On success the code is single-use: it rotates.
func (p *Pairing) Verify(input string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	now := time.Now()
	if now.Before(p.lockedUntil) {
		return ErrLocked
	}
	if now.After(p.expiresAt) {
		p.rotateLocked()
		return ErrExpired
	}
	if input != p.code {
		p.attempts++
		if p.attempts >= p.maxAttempts {
			p.lockedUntil = now.Add(p.lockDur)
			p.attempts = 0
			return ErrLocked
		}
		return ErrInvalidCode
	}
	// Success: single-use, rotate for the next pairing.
	p.rotateLocked()
	return nil
}

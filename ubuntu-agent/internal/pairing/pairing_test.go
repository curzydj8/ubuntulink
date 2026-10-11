package pairing

import (
	"testing"
	"time"
)

func TestVerifySuccess(t *testing.T) {
	p := New(5*time.Minute, 5, 5*time.Minute)
	code := p.Code()
	if err := p.Verify(code); err != nil {
		t.Fatalf("valid code rejected: %v", err)
	}
	// Single-use: same code must now fail.
	if err := p.Verify(code); err == nil {
		t.Fatal("reused code accepted")
	}
}

func TestVerifyWrongCode(t *testing.T) {
	p := New(5*time.Minute, 5, 5*time.Minute)
	if err := p.Verify("000000"); err != ErrInvalidCode {
		t.Fatalf("expected ErrInvalidCode, got %v", err)
	}
}

func TestLockoutAfterMaxAttempts(t *testing.T) {
	p := New(5*time.Minute, 3, 5*time.Minute)
	p.Verify("111111")
	p.Verify("222222")
	if err := p.Verify("333333"); err != ErrLocked {
		t.Fatalf("expected ErrLocked on 3rd attempt, got %v", err)
	}
	// Even the right code is locked out.
	if err := p.Verify(p.Code()); err != ErrLocked {
		t.Fatalf("expected ErrLocked while locked, got %v", err)
	}
}

func TestExpiry(t *testing.T) {
	p := New(50*time.Millisecond, 5, 5*time.Minute)
	code := p.Code()
	time.Sleep(80 * time.Millisecond)
	if err := p.Verify(code); err != ErrExpired {
		t.Fatalf("expected ErrExpired, got %v", err)
	}
	// Expired code rotated: a fresh code exists.
	if p.Code() == code {
		t.Fatal("code did not rotate after expiry")
	}
}

func TestCodeFormat(t *testing.T) {
	p := New(5*time.Minute, 5, 5*time.Minute)
	c := p.Code()
	if len(c) != 6 {
		t.Fatalf("code length = %d, want 6", len(c))
	}
	for _, ch := range c {
		if ch < '0' || ch > '9' {
			t.Fatalf("non-digit in code %q", c)
		}
	}
}

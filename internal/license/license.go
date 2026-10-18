// Package license verifies offline license keys. A key is
// base64url(payload JSON) + "." + base64url(ed25519 signature) signed with the
// vendor's private key; the matching public key is compiled into the binary,
// so the server never needs to call home.
package license

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Payload is the signed content.
type Payload struct {
	Org      string    `json:"org"`
	Seats    int       `json:"seats"`     // 0 = unlimited
	IssuedAt time.Time `json:"issued_at"` //nolint:tagliatelle
	Expires  time.Time `json:"expires"`   // zero = perpetual
}

// License is a verified key plus how it is being applied.
type License struct {
	Payload   Payload
	FreeSeats int // seats allowed with no key at all
	now       func() time.Time
}

var (
	ErrMalformed = errors.New("license: malformed key")
	ErrSignature = errors.New("license: bad signature")
)

// Parse verifies key against pub. An empty key yields a License that
// allows freeSeats (the community tier).
func Parse(key string, pub ed25519.PublicKey, freeSeats int) (*License, error) {
	l := &License{FreeSeats: freeSeats, now: time.Now}
	key = strings.TrimSpace(key)
	if key == "" {
		return l, nil
	}
	parts := strings.Split(key, ".")
	if len(parts) != 2 {
		return nil, ErrMalformed
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrMalformed
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrMalformed
	}
	if len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, payload, sig) {
		return nil, ErrSignature
	}
	if err := json.Unmarshal(payload, &l.Payload); err != nil {
		return nil, ErrMalformed
	}
	return l, nil
}

// Seats implements app.Licensor: allowed active human users and a status line.
// An expired key falls back to the free tier rather than locking anyone out.
func (l *License) Seats() (int, string) {
	p := l.Payload
	if p.IssuedAt.IsZero() && p.Org == "" {
		return l.FreeSeats, fmt.Sprintf("community edition (%d seats)", l.FreeSeats)
	}
	if !p.Expires.IsZero() && l.now().After(p.Expires) {
		return l.FreeSeats, fmt.Sprintf("license for %s expired on %s; community limits apply", p.Org, p.Expires.Format("2006-01-02"))
	}
	if p.Seats <= 0 {
		return 0, fmt.Sprintf("licensed to %s, unlimited seats", p.Org)
	}
	until := "perpetual"
	if !p.Expires.IsZero() {
		until = "until " + p.Expires.Format("2006-01-02")
	}
	return p.Seats, fmt.Sprintf("licensed to %s, %d seats, %s", p.Org, p.Seats, until)
}

// Sign creates a key for payload with the vendor's private key (used by cmd/licensegen).
func Sign(p Payload, priv ed25519.PrivateKey) (string, error) {
	body, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(priv, body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// DecodePublicKey parses a base64url-encoded ed25519 public key.
func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil || len(b) != ed25519.PublicKeySize {
		return nil, errors.New("license: bad public key")
	}
	return ed25519.PublicKey(b), nil
}

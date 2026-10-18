package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"strings"
	"testing"
	"time"
)

func TestSignAndParse(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := Sign(Payload{Org: "ACME", Seats: 50, IssuedAt: time.Now(), Expires: time.Now().Add(24 * time.Hour)}, priv)
	if err != nil {
		t.Fatal(err)
	}
	l, err := Parse(key, pub, 10)
	if err != nil {
		t.Fatal(err)
	}
	seats, status := l.Seats()
	if seats != 50 || !strings.Contains(status, "ACME") {
		t.Fatalf("got %d %q", seats, status)
	}

	// Tampering with the payload must fail the signature.
	tampered := strings.Replace(key, key[:4], "AAAA", 1)
	if _, err := Parse(tampered, pub, 10); err == nil {
		t.Fatal("tampered key accepted")
	}

	// Wrong public key must fail.
	otherPub, _, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Parse(key, otherPub, 10); err == nil {
		t.Fatal("key accepted by the wrong public key")
	}
}

func TestExpiredFallsBackToFreeSeats(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	key, _ := Sign(Payload{Org: "ACME", Seats: 50, IssuedAt: time.Now().Add(-48 * time.Hour), Expires: time.Now().Add(-time.Hour)}, priv)
	l, err := Parse(key, pub, 10)
	if err != nil {
		t.Fatal(err)
	}
	seats, status := l.Seats()
	if seats != 10 || !strings.Contains(status, "expired") {
		t.Fatalf("got %d %q", seats, status)
	}
}

func TestEmptyKeyIsCommunity(t *testing.T) {
	l, err := Parse("", nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if seats, _ := l.Seats(); seats != 10 {
		t.Fatalf("got %d", seats)
	}
}

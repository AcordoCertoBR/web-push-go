package webpush

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestNewVapidIsUsableForSigning(t *testing.T) {
	vapid, err := NewVapid("dev@example.com")
	if err != nil {
		t.Fatalf("NewVapid returned error: %v", err)
	}

	if !vapid.IsValid() {
		t.Fatal("expected freshly generated Vapid to be valid")
	}

	if _, err := vapid.ECDSAPrivKey(); err != nil {
		t.Fatalf("ECDSAPrivKey returned error: %v", err)
	}

	token, err := GetVAPIDAuthorizationHeader("https://push.example.com/send/abc", vapid, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("failed to sign VAPID JWT with freshly generated keys: %v", err)
	}
	if token == "" {
		t.Fatal("expected a non-empty signed token")
	}
}

func TestLoadVapidAcceptsPaddedAndStandardBase64(t *testing.T) {
	original, err := NewVapid("dev@example.com")
	if err != nil {
		t.Fatalf("NewVapid returned error: %v", err)
	}
	privRaw, pubRaw := original.Keys()

	privBytes, err := base64.RawURLEncoding.DecodeString(privRaw)
	if err != nil {
		t.Fatalf("failed to decode private key: %v", err)
	}
	pubBytes, err := base64.RawURLEncoding.DecodeString(pubRaw)
	if err != nil {
		t.Fatalf("failed to decode public key: %v", err)
	}

	// Same keys re-encoded as padded standard base64, as stored by other libraries.
	loaded, err := LoadVapid(
		"dev@example.com",
		base64.StdEncoding.EncodeToString(privBytes),
		base64.StdEncoding.EncodeToString(pubBytes),
	)
	if err != nil {
		t.Fatalf("LoadVapid rejected padded standard base64 keys: %v", err)
	}

	gotPriv, gotPub := loaded.Keys()
	if gotPriv != privRaw || gotPub != pubRaw {
		t.Fatal("expected keys to be normalized back to raw URL-safe base64")
	}
}

func TestLoadVapidRejectsMismatchedPublicKey(t *testing.T) {
	a, err := NewVapid("dev@example.com")
	if err != nil {
		t.Fatalf("NewVapid returned error: %v", err)
	}
	b, err := NewVapid("dev@example.com")
	if err != nil {
		t.Fatalf("NewVapid returned error: %v", err)
	}

	privA, _ := a.Keys()
	_, pubB := b.Keys()

	if _, err := LoadVapid("dev@example.com", privA, pubB); err == nil {
		t.Fatal("expected error when public key does not match private key")
	}
}

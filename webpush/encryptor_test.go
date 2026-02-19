package webpush

import (
	"encoding/base64"
	"encoding/binary"
	"testing"
)

func TestEncryptPayload_DefaultRecordLayout(t *testing.T) {
	_, serverPriv, _, err := generateServerKeys()
	if err != nil {
		t.Fatalf("failed to generate keys: %v", err)
	}

	clientPub := serverPriv.PublicKey().Bytes()
	p256dh := base64.RawURLEncoding.EncodeToString(clientPub)
	auth := base64.RawURLEncoding.EncodeToString([]byte("0123456789ABCDEF"))

	payload := "{\"title\":\"hello\",\"body\":\"world\"}"
	result, err := EncryptPayload(p256dh, auth, payload, 0)
	if err != nil {
		t.Fatalf("EncryptPayload returned error: %v", err)
	}

	if len(result) <= 16+4+1 {
		t.Fatalf("encrypted payload too short: %d", len(result))
	}

	recordSize := binary.BigEndian.Uint32(result[16:20])
	if recordSize != MaxRecordSize {
		t.Fatalf("expected record size %d, got %d", MaxRecordSize, recordSize)
	}

	serverKeyLen := int(result[20])
	if serverKeyLen != len(serverPriv.PublicKey().Bytes()) {
		t.Fatalf("unexpected server public key len: %d", serverKeyLen)
	}

	ciphertext := result[21+serverKeyLen:]
	if len(ciphertext) == 0 {
		t.Fatal("ciphertext is empty")
	}
}

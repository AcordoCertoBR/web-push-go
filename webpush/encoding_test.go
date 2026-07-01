package webpush

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeBase64KeyVariants(t *testing.T) {
	// Bytes chosen so URL-safe and standard alphabets actually differ (-_ vs +/).
	raw := []byte{0xfb, 0xef, 0xbe, 0xff, 0x01, 0x02}

	tests := []struct {
		name    string
		encoded string
	}{
		{name: "raw url-safe", encoded: base64.RawURLEncoding.EncodeToString(raw)},
		{name: "padded url-safe", encoded: base64.URLEncoding.EncodeToString(raw)},
		{name: "raw standard", encoded: base64.RawStdEncoding.EncodeToString(raw)},
		{name: "padded standard", encoded: base64.StdEncoding.EncodeToString(raw)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeBase64Key(tt.encoded)
			if err != nil {
				t.Fatalf("decodeBase64Key(%q) returned error: %v", tt.encoded, err)
			}
			if !bytes.Equal(got, raw) {
				t.Fatalf("decodeBase64Key(%q) = %x, want %x", tt.encoded, got, raw)
			}
		})
	}
}

func TestDecodeBase64KeyRejectsInvalidInput(t *testing.T) {
	if _, err := decodeBase64Key("not base64 at all!!"); err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
}

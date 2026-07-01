package webpush

import (
	"encoding/base64"
	"errors"
	"strings"
)

var errInvalidBase64Key = errors.New("key is not valid base64 (raw or padded, url-safe or standard)")

// decodeBase64Key decodes a base64-encoded key accepting the encoding
// variants found in the wild: raw URL-safe (per spec), padded URL-safe, and
// standard base64 with or without padding. Subscriptions and VAPID keys
// stored by other libraries or older systems are not always raw URL-safe.
func decodeBase64Key(key string) ([]byte, error) {
	key = strings.TrimRight(key, "=")

	if b, err := base64.RawURLEncoding.DecodeString(key); err == nil {
		return b, nil
	}
	if b, err := base64.RawStdEncoding.DecodeString(key); err == nil {
		return b, nil
	}

	return nil, errInvalidBase64Key
}

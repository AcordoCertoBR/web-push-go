package webpush

import (
	"bytes"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"math/big"
)

type Vapid interface {
	// IsValid checks if the VAPID information is valid.
	IsValid() bool
	// Keys returns the VAPID keys: private ECDH and public ECDH.
	Keys() (privateECDH, publicECDH string)
	// Subject returns the VAPID subject (email address or URL).
	Subject() string
	// ECDSAPrivKey returns the ECDSA private key, which is used to sign the VAPID JWT token.
	ECDSAPrivKey() (*ecdsa.PrivateKey, error)
}

type VapidInfo struct {
	// subject (email address or URL)
	subject string
	// ECDH public key (base64 raw URL-safe encoded string)
	pubECDHKey  string
	privECDHKey string
	// ECDSA private key (base64 raw URL-safe encoded string).
	// This key is used to sign the VAPID JWT token and derive private ECDH key if needed.
	// Useful to import the ecdh key pair in other libraries or languages.
	privECDSAKey string
}

// NewVapid creates a Vapid instance with a freshly generated P-256 key pair.
// The subject is typically an email address or URL that identifies the sender.
// The returned instance is fully initialized, including the ECDSA signing key.
func NewVapid(subject string) (Vapid, error) {
	privECDH, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}

	return LoadVapid(
		subject,
		base64.RawURLEncoding.EncodeToString(privECDH.Bytes()),
		base64.RawURLEncoding.EncodeToString(privECDH.PublicKey().Bytes()),
	)
}

// LoadVapid creates a new Vapid instance with the provided keys and subject.
// This function is useful for loading existing VAPID information from storage or configuration.
//
// Keys are accepted in raw or padded base64, URL-safe or standard, and are
// normalized internally to raw URL-safe (the format required on the wire).
// The provided public key must match the one derived from the private key.
func LoadVapid(subject, privECDH, pubECDH string) (Vapid, error) {
	privBytes, err := decodeBase64Key(privECDH)
	if err != nil {
		return nil, err
	}

	ecdhPriv, err := ecdh.P256().NewPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	pubBytes := ecdhPriv.PublicKey().Bytes()

	providedPubBytes, err := decodeBase64Key(pubECDH)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(pubBytes, providedPubBytes) {
		return nil, errors.New("public key does not match the provided private key")
	}

	ecdsaKey := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(pubBytes[1:33]),
			Y:     new(big.Int).SetBytes(pubBytes[33:]),
		},
		D: new(big.Int).SetBytes(privBytes),
	}

	ecdsaKeyBytes, err := x509.MarshalECPrivateKey(ecdsaKey)
	if err != nil {
		return nil, err
	}

	return &VapidInfo{
		subject:      subject,
		pubECDHKey:   base64.RawURLEncoding.EncodeToString(pubBytes),
		privECDHKey:  base64.RawURLEncoding.EncodeToString(privBytes),
		privECDSAKey: base64.RawURLEncoding.EncodeToString(ecdsaKeyBytes),
	}, nil
}

func (v VapidInfo) IsValid() bool {
	return v.subject != "" && v.pubECDHKey != "" && v.privECDHKey != "" && v.privECDSAKey != ""
}

func (v VapidInfo) Subject() (subject string) {
	return v.subject
}

func (v VapidInfo) Keys() (privateECDH, publicECDH string) {
	return v.privECDHKey, v.pubECDHKey
}

func (v VapidInfo) ECDSAPrivKey() (*ecdsa.PrivateKey, error) {
	b, err := base64.RawURLEncoding.DecodeString(v.privECDSAKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseECPrivateKey(b)
}

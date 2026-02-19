package webpush

import (
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"fmt"
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
	// ECDH public key (base64 encoded string)
	pubECDHKey  string
	privECDHKey string
	// ECDSA private key (base64 encoded string).
	// This key is used to sign the VAPID JWT token and derive private ECDH key if needed.
	// Useful to import the ecdh key pair in other libraries or languages.
	privECDSAKey string
}

// Create a new Vapid instance with ECDH private and public key.
// The subject is typically an email address or URL that identifies the sender.
// Returns the Vapid instance and an error if any occurs during key generation.
// If the keys are not valid, the Vapid instance will have empty fields.
func NewVapid(subject string) (vapid Vapid, err error) {
	privECDH, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return vapid, err
	}
	pubECDH := privECDH.PublicKey()

	vapid = &VapidInfo{
		subject:     subject,
		pubECDHKey:  base64.RawURLEncoding.EncodeToString(pubECDH.Bytes()),
		privECDHKey: base64.RawURLEncoding.EncodeToString(privECDH.Bytes()),
	}

	return vapid, nil
}

// LoadVapid creates a new Vapid instance with the provided keys and subject.
// This function is useful for loading existing VAPID information from storage or configuration.
func LoadVapid(subject, privECDH, pubECDH string) (Vapid, error) {
	privBytes, err := base64.RawURLEncoding.DecodeString(privECDH)
	if err != nil {
		return nil, err
	}

	ecdhPriv, err := ecdh.P256().NewPrivateKey(privBytes)
	if err != nil {
		return nil, err
	}

	ecdhPub := ecdhPriv.PublicKey()
	pubBytes := ecdhPub.Bytes()

	if len(pubBytes) != 65 {
		return nil, fmt.Errorf("invalid public key length")
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
		pubECDHKey:   pubECDH,
		privECDHKey:  privECDH,
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

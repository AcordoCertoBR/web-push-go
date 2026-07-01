package webpush

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"

	"errors"
)

var ErrMaxPadExceeded = errors.New("payload has exceeded the maximum length")

const (
	MaxRecordSize uint32 = 4096
)

// EncryptPayload encrypts a Web Push message payload using the subscriber's P-256 public key
// (p256dh) and authentication secret (auth), returning the final aes128gcm-encrypted body
// formatted per the Web Push payload structure.
//
// The function:
//   - decodes the URL-safe base64 subscriber keys,
//   - generates an ephemeral server P-256 key pair,
//   - derives a shared secret via ECDH,
//   - derives IKM, content-encryption key (CEK), and nonce using HKDF-SHA256,
//   - appends the Web Push record delimiter byte (0x02) to the payload,
//   - encrypts with AES-128-GCM,
//   - and serializes the result as: salt || recordSize || keyLen || serverPubKey || ciphertext.
//
// If customRecordSize is 0, MaxRecordSize is used. If payload size plus delimiter and GCM tag
// exceeds the selected record size, ErrMaxPadExceeded is returned.
//
// Parameters:
//   - p256dh: URL-safe base64 (raw, no padding) encoded client P-256 public key.
//   - auth: URL-safe base64 (raw, no padding) encoded authentication secret.
//   - payload: plaintext message to encrypt.
//   - customRecordSize: optional record size override; 0 means MaxRecordSize.
//
// Returns:
//   - encryptedBody: serialized encrypted payload body ready to send in a Web Push request.
//   - err: non-nil on key decoding, key derivation, randomness, cipher setup, or size validation failure.
func EncryptPayload(p256dh, auth, payload string, customRecordSize uint32) (encryptedBody []byte, err error) {
	clientPubKeyBytes, err := decodeBase64Key(p256dh)
	if err != nil {
		return encryptedBody, fmt.Errorf("%w: p256dh: %v", ErrInvalidSubscription, err)
	}

	authSecret, err := decodeBase64Key(auth)
	if err != nil {
		return encryptedBody, fmt.Errorf("%w: auth: %v", ErrInvalidSubscription, err)
	}

	curve, serverPrivKey, serverPubKey, err := generateServerKeys()
	if err != nil {
		return encryptedBody, err
	}

	clientPubKey, err := curve.NewPublicKey(clientPubKeyBytes)
	if err != nil {
		return encryptedBody, err
	}

	derivedKey, err := serverPrivKey.ECDH(clientPubKey)
	if err != nil {
		return encryptedBody, err
	}

	authInfo := append([]byte("WebPush: info\x00"), clientPubKeyBytes...)
	authInfo = append(authInfo, serverPubKey.Bytes()...)
	ikm, err := hkdf.Key(sha256.New, derivedKey, authSecret, string(authInfo), 32)
	if err != nil {
		return encryptedBody, err
	}

	salt := make([]byte, 16)
	_, err = rand.Read(salt)
	if err != nil {
		return encryptedBody, err
	}

	contentEncryptionKeyInfo := "Content-Encoding: aes128gcm\x00"
	cek, err := hkdf.Key(sha256.New, ikm, salt, contentEncryptionKeyInfo, 16)
	if err != nil {
		return encryptedBody, err
	}

	nonceInfo := "Content-Encoding: nonce\x00"
	nonce, err := hkdf.Key(sha256.New, ikm, salt, nonceInfo, 12)
	if err != nil {
		return encryptedBody, err
	}

	recordSize := customRecordSize
	if recordSize == 0 {
		recordSize = MaxRecordSize
	}

	if uint32(len(payload)+1+16) > recordSize {
		return encryptedBody, ErrMaxPadExceeded
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return encryptedBody, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return encryptedBody, err
	}

	plaintext := append([]byte(payload), 0x02)
	ciphertext := aesgcm.Seal([]byte{}, nonce, plaintext, nil)

	encodedRecordSize := make([]byte, 4)
	binary.BigEndian.PutUint32(encodedRecordSize, recordSize)

	body := make([]byte, 0, 16+4+1+len(serverPubKey.Bytes())+len(ciphertext))
	body = append(body, salt...)
	body = append(body, encodedRecordSize...)
	body = append(body, byte(len(serverPubKey.Bytes())))
	body = append(body, serverPubKey.Bytes()...)
	body = append(body, ciphertext...)

	return body, nil
}

func generateServerKeys() (curve ecdh.Curve, privateKey *ecdh.PrivateKey, publicKey *ecdh.PublicKey, err error) {
	curve = ecdh.P256()
	privateKey, err = curve.GenerateKey(rand.Reader)
	if err != nil {
		return curve, privateKey, publicKey, err
	}
	publicKey = privateKey.PublicKey()

	return curve, privateKey, publicKey, nil
}

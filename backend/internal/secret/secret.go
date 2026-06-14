// Package secret seals and opens sensitive data (e.g. device credentials) at
// rest using AES-256-GCM. The key is derived from NEKO_SECRET_KEY; in
// production this MUST be a strong, persistent secret (the deploy script
// generates one). Never log plaintext or keys.
package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

// Sealer encrypts/decrypts small blobs with a fixed key.
type Sealer struct {
	gcm cipher.AEAD
}

// New derives a 256-bit key from the given passphrase (SHA-256) and returns a
// Sealer. An empty passphrase falls back to a fixed dev key (insecure).
func New(passphrase string) (*Sealer, error) {
	if passphrase == "" {
		passphrase = "neko-insecure-dev-key-change-me"
	}
	key := sha256.Sum256([]byte(passphrase))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Sealer{gcm: gcm}, nil
}

// Seal encrypts plaintext and returns a base64 string (nonce||ciphertext).
func (s *Sealer) Seal(plaintext []byte) (string, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := s.gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

// Open decrypts a base64 string produced by Seal.
func (s *Sealer) Open(enc string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil, err
	}
	ns := s.gcm.NonceSize()
	if len(raw) < ns {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ct := raw[:ns], raw[ns:]
	return s.gcm.Open(nil, nonce, ct, nil)
}

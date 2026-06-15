package routing

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateWGKeyPair returns a WireGuard Curve25519 private/public key pair
// encoded in standard base64 (RouterOS wireguard format).
func GenerateWGKeyPair() (privateKey, publicKey string, err error) {
	var sk [32]byte
	if _, err := rand.Read(sk[:]); err != nil {
		return "", "", fmt.Errorf("rand: %w", err)
	}
	// WireGuard private-key clamping (RFC 7748).
	sk[0] &= 248
	sk[31] &= 127
	sk[31] |= 64
	var pk [32]byte
	curve25519.ScalarBaseMult(&pk, &sk)
	return base64.StdEncoding.EncodeToString(sk[:]),
		base64.StdEncoding.EncodeToString(pk[:]),
		nil
}

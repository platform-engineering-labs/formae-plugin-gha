// © 2025 Platform Engineering Labs Inc.
//
// SPDX-License-Identifier: FSL-1.1-ALv2

package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/nacl/box"
)

// HashSecret returns the SHA-256 hex digest of a plaintext secret value.
// Used for drift detection since the GitHub API never returns secret values.
func HashSecret(plaintext string) string {
	h := sha256.Sum256([]byte(plaintext))
	return hex.EncodeToString(h[:])
}

// EncryptSecret encrypts a plaintext value using the repository/org/env public key
// via libsodium sealed-box (X25519-XSalsa20-Poly1305).
// The publicKeyB64 is the base64-encoded public key from the GitHub API.
func EncryptSecret(plaintext string, publicKeyB64 string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyB64)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	if len(publicKeyBytes) != 32 {
		return "", fmt.Errorf("invalid public key length: expected 32, got %d", len(publicKeyBytes))
	}

	var recipientKey [32]byte
	copy(recipientKey[:], publicKeyBytes)

	encrypted, err := box.SealAnonymous(nil, []byte(plaintext), &recipientKey, rand.Reader)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt secret: %w", err)
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

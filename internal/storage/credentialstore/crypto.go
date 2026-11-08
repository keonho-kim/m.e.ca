// Purpose: Generate a fresh AES-256-GCM key per saved credential version and authenticate record identity.
// Pattern: Authenticated encryption with an independent durable key store.
// Usage: Store.Put verifies OS persistence before publishing ciphertext; Store.Get never recreates missing keys.
// Related: internal/storage/credentialstore/store.go, internal/config/credentials/keyring.go.
package credentialstore

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"

	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func aad(ref, scope, keyRef string) []byte {
	return []byte("meca-credential-aes256gcm-v1\x00" + ref + "\x00" + scope + "\x00" + keyRef)
}

func (s *Store) seal(ctx context.Context, ref, scope, secret string) (keyRef string, sealed []byte, resultErr error) {
	reservation, err := s.reserveKey(ctx)
	if err != nil {
		return "", nil, err
	}
	defer func() { resultErr = errors.Join(resultErr, reservation.Close()) }()
	var material [32]byte
	if _, err := rand.Read(material[:]); err != nil {
		return "", nil, err
	}
	defer clear(material[:])
	keyRef = "encryption_" + rand.Text()
	encoded := base64.StdEncoding.EncodeToString(material[:])
	reservation.Publish()
	if err := s.keys.Set(ctx, keyRef, encoded); err != nil {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		return "", nil, ErrKeyUnavailable
	}
	verified, err := s.keys.Get(ctx, keyRef)
	if err != nil || subtle.ConstantTimeCompare([]byte(encoded), []byte(verified)) != 1 {
		if ctx.Err() != nil {
			return "", nil, ctx.Err()
		}
		return "", nil, ErrKeyUnavailable
	}
	block, err := aes.NewCipher(material[:])
	if err != nil {
		return "", nil, err
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return "", nil, err
	}
	return keyRef, aead.Seal(nil, nil, []byte(secret), aad(ref, scope, keyRef)), nil
}

func (s *Store) open(ctx context.Context, ref, scope, keyRef string, sealed []byte) (string, error) {
	if len(keyRef) != 37 || !strings.HasPrefix(keyRef, "encryption_") || len(sealed) < 29 || len(sealed) > 2048+28 {
		return "", ErrIntegrity
	}
	encoded, err := s.keys.Get(ctx, keyRef)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", ErrKeyUnavailable
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if len(encoded) != base64.StdEncoding.EncodedLen(32) {
		return "", ErrKeyUnavailable
	}
	material, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil || len(material) != 32 {
		return "", ErrKeyUnavailable
	}
	defer clear(material)
	block, err := aes.NewCipher(material)
	if err != nil {
		return "", faults.Errorf(diagcatalog.CredentialstoreInvalidEncryptionKey)
	}
	aead, err := cipher.NewGCMWithRandomNonce(block)
	if err != nil {
		return "", err
	}
	plain, err := aead.Open(nil, nil, sealed, aad(ref, scope, keyRef))
	if err != nil {
		return "", ErrIntegrity
	}
	defer clear(plain)
	return string(plain), nil
}

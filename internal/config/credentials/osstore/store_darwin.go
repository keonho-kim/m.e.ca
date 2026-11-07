// Purpose: Store encryption keys in macOS Keychain with cancellable, joined security commands.
// Pattern: Native OS credential adapter.
// Usage: Vault uses Store with MECA's fixed service; secrets are passed through stdin only.
// Related: internal/config/credentials/keyring.go, internal/storage/credentialstore/crypto.go.
package osstore

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Store has no persistent processes or handles. Each method owns its command.
type Store struct {
	Service string
	run     func(context.Context, string, ...string) ([]byte, error)
}

// This is the current encoding of OS entries, including already saved AES keys.
const keychainEncoding = "go-keyring-base64:"

func keychainIdentifier(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '_' || r == '-' || r == '.') {
			return false
		}
	}
	return true
}

func (s Store) command(ctx context.Context, input string, args ...string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.run != nil {
		return s.run(ctx, input, args...)
	}
	command := exec.CommandContext(ctx, "/usr/bin/security", args...)
	command.Stdin = strings.NewReader(input)
	command.WaitDelay = time.Second
	body, err := command.Output()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return body, err
}

func (s Store) Set(ctx context.Context, ref, value string) error {
	if !keychainIdentifier(s.Service) || !keychainIdentifier(ref) {
		return fmt.Errorf("invalid OS credential identity")
	}
	encoded := keychainEncoding + base64.StdEncoding.EncodeToString([]byte(value))
	input := fmt.Sprintf("add-generic-password -U -s %s -a %s -w %s\n", s.Service, ref, encoded)
	_, err := s.command(ctx, input, "-i")
	return err
}

func (s Store) Get(ctx context.Context, ref string) (string, error) {
	if !keychainIdentifier(s.Service) || !keychainIdentifier(ref) {
		return "", fmt.Errorf("invalid OS credential identity")
	}
	body, err := s.command(ctx, "", "find-generic-password", "-s", s.Service, "-wa", ref)
	if keychainMissing(err) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(string(body))
	if !strings.HasPrefix(text, keychainEncoding) {
		return "", fmt.Errorf("unsupported OS encryption key encoding")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(text, keychainEncoding))
	if err != nil {
		return "", fmt.Errorf("invalid OS encryption key encoding")
	}
	defer clear(decoded)
	return string(decoded), nil
}

func (s Store) Clear(ctx context.Context) error {
	if !keychainIdentifier(s.Service) {
		return fmt.Errorf("invalid OS credential service")
	}
	for {
		_, err := s.command(ctx, "", "delete-generic-password", "-s", s.Service)
		if keychainMissing(err) {
			return nil
		}
		if err != nil {
			return err
		}
	}
}

func keychainMissing(err error) bool {
	var exit interface {
		error
		ExitCode() int
	}
	return errors.As(err, &exit) && exit.ExitCode() == 44 // errSecItemNotFound (-25300).
}

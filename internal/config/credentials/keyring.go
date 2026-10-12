// Purpose: Store provider keys in the OS credential store and sanitize secrets and terminal controls.
// Pattern: Credential Adapter.
// Usage: Used by model selection and task output; no plaintext fallback is written.
// Related: internal/provider/connection/connection.go, internal/agent/run/run.go.
package credentials

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"

	"github.com/zalando/go-keyring"
)

const service = "v-code"

type Vault struct {
	mu   sync.RWMutex
	keys []string
}

func (v *Vault) Remember(key string) {
	if key == "" {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, old := range v.keys {
		if old == key {
			return
		}
	}
	v.keys = append(v.keys, key)
}
func (v *Vault) Set(ref, key string) error {
	if ref == "" || len(key) > 2048 || strings.ContainsAny(key, "\r\n\x00") {
		return fmt.Errorf("invalid credential")
	}
	v.Remember(key)
	if key == "" {
		err := keyring.Delete(service, ref)
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return err
	}
	if err := keyring.Set(service, ref, key); err != nil {
		return fmt.Errorf("OS credential store unavailable; key was not saved")
	}
	return nil
}
func (v *Vault) Get(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	key, err := keyring.Get(service, ref)
	if err != nil {
		return "", fmt.Errorf("credential unavailable; reconnect using !model")
	}
	v.Remember(key)
	return key, nil
}
func (v *Vault) Redact(text string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	for _, key := range v.keys {
		text = strings.ReplaceAll(text, key, "[redacted]")

	}
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
}

// Stream receives the accumulated response, not isolated transport chunks.
func (v *Vault) Stream(text string) string {
	v.mu.RLock()
	for _, key := range v.keys {
		// Hide incomplete key prefixes at a streaming boundary as well.
		for n := min(len(key)-1, len(text)); n > 0; n-- {
			if strings.HasSuffix(text, key[:n]) {
				text = text[:len(text)-n] + "[redacted]"
				break
			}
		}
	}
	v.mu.RUnlock()
	return v.Redact(text)
}

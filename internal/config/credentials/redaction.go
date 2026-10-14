// Purpose: Remember active secrets and mask complete and partial streamed values and terminal controls.
// Pattern: Secret sanitization boundary.
// Usage: Provider selection remembers keys and runtime output calls Vault.Redact or Stream.
// Related: internal/config/credentials/keyring.go, internal/agent/run/generate.go.
package credentials

import (
	"strings"
	"sync"
	"unicode"
)

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

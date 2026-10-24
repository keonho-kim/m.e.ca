// Purpose: Remember active secrets and sanitize complete text values and terminal controls.
// Pattern: Secret sanitization boundary.
// Usage: Provider selection remembers keys; complete values use Redact and streamed values use NewStreamMask.
// Related: internal/config/credentials/keyring.go, internal/agent/run/generate.go.
package credentials

import (
	"strings"
	"sync"
	"unicode"
)

type Vault struct {
	State string
	mu    sync.RWMutex
	keys  []string
}

func (v *Vault) RedactionVersion() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.keys)
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

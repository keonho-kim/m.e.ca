// Purpose: Remember active secrets and sanitize complete text values and terminal controls.
// Pattern: Secret sanitization boundary.
// Usage: Provider selection remembers keys; complete values use Redact and streamed values use NewStreamMask.
// Related: internal/config/credentials/vault.go, internal/agent/run/generate.go.
package redaction

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

type Registry struct {
	mu       sync.RWMutex
	keys     []string
	index    *keyIndex
	replacer *strings.Replacer
}

func (v *Registry) RedactionVersion() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.keys)
}

func (v *Registry) Remember(key string) error {
	key = strings.Trim(key, " \t")
	if key == "" {
		return nil
	}
	if len(key) > 2048 || !utf8.ValidString(key) || strings.ContainsFunc(key, unicode.IsControl) {
		return fmt.Errorf("invalid credential for private masking")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, old := range v.keys {
		if old == key {
			return nil
		}
	}
	if len(v.keys) >= 1024 {
		return fmt.Errorf("credential masking history limit reached; restart MECA before selecting another key")
	}
	v.keys = append(v.keys, key)
	v.index = indexKeys(v.keys)
	// Build a fresh immutable matcher. Existing readers may still own the old
	// snapshot, and the longest original match must win before any replacement.
	ordered := append([]string(nil), v.keys...)
	sort.Slice(ordered, func(i, j int) bool { return len(ordered[i]) > len(ordered[j]) })
	pairs := make([]string, 0, 2*len(ordered))
	for _, value := range ordered {
		pairs = append(pairs, value, "[redacted]")
	}
	v.replacer = strings.NewReplacer(pairs...)
	return nil
}
func (v *Registry) Redact(text string) string {
	v.mu.RLock()
	replacer := v.replacer
	v.mu.RUnlock()
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, text)
	if replacer != nil {
		return replacer.Replace(text)
	}
	return text
}

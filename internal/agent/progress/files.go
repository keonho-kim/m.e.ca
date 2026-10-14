// Purpose: Detect file content cycles while resetting evidence after external changes.
// Pattern: Pure state transition.
// Usage: The runtime observes committed file changes through Files.Repeated.
// Related: internal/agent/tool/execution.go.
package progress

import (
	"crypto/sha256"
	"encoding/json"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

type Files struct {
	current map[string]string
	seen    map[[32]byte]bool
}

func (s *Files) Repeated(changes []toolapi.FileChange) bool {
	if s.current == nil {
		s.current = map[string]string{}
	}
	reset := false
	observed := false
	for _, change := range changes {
		if change.BeforeDigest == "" || change.AfterDigest == "" {
			continue
		}
		observed = true
		if old, ok := s.current[change.Path]; !ok || old != change.BeforeDigest {
			reset = true
			s.current[change.Path] = change.BeforeDigest
		}
		if change.Destination != "" {
			if _, ok := s.current[change.Destination]; !ok {
				s.current[change.Destination] = "missing"
				reset = true
			}
		}
	}
	if !observed {
		return false
	}
	if reset || s.seen == nil {
		s.seen = map[[32]byte]bool{}
	}
	// JSON gives deterministic key ordering for this string map.
	before, _ := json.Marshal(s.current)
	s.seen[sha256.Sum256(before)] = true
	for _, change := range changes {
		if change.BeforeDigest == "" || change.AfterDigest == "" {
			continue
		}
		s.current[change.Path] = change.AfterDigest
		if change.Destination != "" {
			s.current[change.Destination] = change.BeforeDigest
		}
	}
	after, _ := json.Marshal(s.current)
	fingerprint := sha256.Sum256(after)
	repeated := s.seen[fingerprint]
	s.seen[fingerprint] = true
	return repeated
}

// Purpose: Retain semantic observations for one request.
// Pattern: Request-owned state.
// Usage: The run owner compares tool results without retaining their full bodies.
// Related: internal/agent/run/observation.go, internal/agent/run/checkpoint.go.
package progress

import (
	"crypto/sha256"
	"encoding/hex"
)

type Request struct {
	seen map[string]bool
}

func (r *Request) Observe(key string) bool {
	if r.seen == nil {
		r.seen = map[string]bool{}
	}
	digest := sha256.Sum256([]byte(key))
	id := hex.EncodeToString(digest[:])
	if r.seen[id] {
		return false
	}
	r.seen[id] = true
	return true
}

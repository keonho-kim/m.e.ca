// Purpose: Preserve explicit user refusals across peers for the same frozen operation.
// Pattern: Request-scoped denial ledger.
// Usage: Approval composition records refusals; local execution checks them before effects.
// Related: internal/tools/team/local.go, internal/commands/teamrequest/request.go.
package approval

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"sync"
	"sync/atomic"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Ledger struct {
	mu        sync.Mutex
	denied    map[string]bool
	populated atomic.Bool
}

func denialKey(proposal toolapi.Approval) string {
	if proposal.Candidate != nil {
		return proposal.Candidate.Binding
	}
	if proposal.Command != nil {
		digest := sha256.New()
		writeString(digest, proposal.Command.Directory)
		digest.Write([]byte{0})
		writeString(digest, proposal.Target)
		var sum [sha256.Size]byte
		digest.Sum(sum[:0])
		return hex.EncodeToString(sum[:])
	}
	return ""
}
func (w *Ledger) Deny(proposal toolapi.Approval) {
	key := denialKey(proposal)
	if key == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.denied == nil {
		w.denied = map[string]bool{}
	}
	w.denied[key] = true
	w.populated.Store(true)
}
func (w *Ledger) Refused(proposal toolapi.Approval, cached *string) bool {
	if w == nil || !w.populated.Load() {
		return false
	}
	key := ""
	if cached != nil {
		key = *cached
	}
	if key == "" {
		key = denialKey(proposal)
		if cached != nil {
			*cached = key
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.denied[key]
}

// Bound conversion storage rather than allocating the entire shell string.
func writeString(digest hash.Hash, text string) {
	var chunk [4096]byte
	for len(text) > 0 {
		n := copy(chunk[:], text)
		digest.Write(chunk[:n])
		text = text[n:]
	}
}

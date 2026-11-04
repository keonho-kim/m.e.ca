// Purpose: Preserve explicit user refusals across peers for the same frozen operation.
// Pattern: Request-scoped denial ledger.
// Usage: Approval composition records refusals; local execution checks them before effects.
// Related: internal/tools/team/local.go, internal/commands/teamrequest/request.go.
package approval

import (
	"crypto/sha256"
	"fmt"
	"sync"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type Ledger struct {
	mu     sync.Mutex
	denied map[string]bool
}

func denialKey(proposal toolapi.Approval) string {
	if proposal.Candidate != nil {
		return proposal.Candidate.Binding
	}
	if proposal.Command != nil {
		digest := sha256.New()
		digest.Write([]byte(proposal.Command.Directory))
		digest.Write([]byte{0})
		digest.Write([]byte(proposal.Target))
		return fmt.Sprintf("%x", digest.Sum(nil))
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
}
func (w *Ledger) Refused(proposal toolapi.Approval) bool {
	key := denialKey(proposal)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.denied[key]
}

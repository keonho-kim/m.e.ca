// Purpose: Preserve explicit user refusals across peers for the same frozen operation.
// Pattern: Request-scoped denial ledger.
// Usage: Approval composition records refusals; local execution checks them before effects.
// Related: internal/tools/team/workspace/workspace.go, internal/commands/teamrequest/request.go.
package workspace

import (
	"crypto/sha256"
	"fmt"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

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
func (w *Access) Deny(proposal toolapi.Approval) {
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
func (w *Access) Refused(proposal toolapi.Approval) bool {
	key := denialKey(proposal)
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.denied[key]
}

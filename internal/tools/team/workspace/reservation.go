// Purpose: Reject a prepared file operation if path aliases changed after workspace admission.
// Pattern: Execution-boundary Lease validation.
// Usage: The cooperative adapter checks the frozen candidate before retaining it for approval.
// Related: internal/tools/team/workspace/workspace.go, internal/tools/team/executor.go.
package workspace

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"os"
)

func (l *Lease) Check(proposal toolapi.Approval) error {
	if l == nil {
		return nil
	}
	w := l.workspace
	w.mu.Lock()
	held := w.writes[l]
	w.mu.Unlock()
	for _, key := range held {
		current := canonical(key.path)
		if current.path != key.path || key.info != nil && current.info != nil && !os.SameFile(key.info, current.info) {
			return changedReservation()
		}
	}
	if proposal.Candidate == nil {
		return nil
	}
	for _, file := range proposal.Candidate.Files {
		actual := canonical(file.Path)
		covered := false
		for _, key := range held {
			if actual.path == key.path || file.AfterDigest == "directory" && overlap(actual, key) {
				covered = true
				break
			}
		}
		if !covered {
			return changedReservation()
		}
	}
	return nil
}
func changedReservation() error {
	return &toolapi.Failure{Code: toolapi.StaleTarget, Detail: "The file target changed while shared access was being prepared; nothing executed.", Next: "Inspect the current target and resubmit the needed operation."}
}

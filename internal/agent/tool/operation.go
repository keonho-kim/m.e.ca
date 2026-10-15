// Purpose: Describe observed tool execution facts independently of compaction.
// Pattern: Domain record.
// Usage: Runtime and compaction preserve Operation records.
// Related: internal/agent/tool/execution.go, internal/agent/plan/step_ref.go.
package tool

import "github.com/keonho-kim/v-code/internal/agent/plan"

type Operation struct {
	Failure                                      *Failure
	Process                                      *ProcessOutcome
	RunID, CallID, Name, Status, Command, Result string
	Check                                        bool
	Step                                         *plan.StepRef
	Changes                                      []FileChange
	Source                                       int64
}

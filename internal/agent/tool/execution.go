// Purpose: Define approval-bound tool execution and observed outcomes.
// Pattern: Consumer-owned execution contract.
// Usage: Local adapters satisfy Executor and Execution; the task loop consumes them.
// Related: internal/tools/local/executor.go, internal/agent/run/execute.go.

package tool

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/policy"
)

type Approval struct {
	Candidate *Candidate
	Command   *CommandApproval
	ID        string
	Name      string
	Target    string
	Preview   string
	Diff      bool
	Required  bool
}

type CommandApproval struct {
	Directory string
	Findings  []CommandFinding
}

type CommandFinding struct {
	Command          string
	Reasons, Targets []string
}

type LineChange struct {
	OldStartLine, OldEndLine, NewStartLine, NewEndLine int
	Removed                                            bool
}

type FileChange struct {
	Lines                     []LineChange `json:",omitempty"`
	Action, Path, Destination string
	BeforeDigest, AfterDigest string
}
type Result struct {
	Capture  json.RawMessage
	Versions map[string]string
	Target   string
	Started  bool
	Failure  *Failure
	Process  *ProcessOutcome
	Content  json.RawMessage
	Summary  string
	Failed   bool
	Changes  []FileChange
}

type Execution interface {
	Approval() Approval
	Execute(context.Context, bool) (Result, error)
}

type Executor interface {
	Definitions() []Definition
	Prepare(context.Context, policy.Settings, Call) (Execution, error)
}

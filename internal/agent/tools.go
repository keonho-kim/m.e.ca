// Purpose: Define the agent-owned tool execution and approval contracts.
// Pattern: Consumer-owned I/O contract.
// Usage: Implemented by tools and consumed by the model/tool loop.
// Related: internal/agent/model.go, internal/tools/local/executor.go.

package agent

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/v-code/internal/policy"
)

type ApprovalRequest struct {
	ID       string
	Name     string
	Target   string
	Preview  string
	Diff     bool
	Required bool
}

type FileChange struct{ Action, Path, Destination string }
type ToolResult struct {
	Content json.RawMessage
	Summary string
	Failed  bool
	Changes []FileChange
}

type ToolExecution interface {
	Approval() ApprovalRequest
	Execute(context.Context, bool) (ToolResult, error)
}

type ToolExecutor interface {
	Definitions() []ToolDefinition
	Prepare(context.Context, policy.Settings, bool, ToolCall) (ToolExecution, error)
}

// ParameterKind is the literal value type requested in one model reply.
type ParameterKind string

const (
	Text        ParameterKind = "text"
	LiteralPath ParameterKind = "literal_path"
	Integer     ParameterKind = "integer"
)

type ToolParameter struct {
	Name, Question       string
	Kind                 ParameterKind
	Required, AllowEmpty bool
	Min, Max, MaxBytes   int
}
type ToolDefinition struct {
	Name, Description string
	Effect            policy.Effect
	Parameters        []ToolParameter
}

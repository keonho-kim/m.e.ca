// Purpose: Define one main-agent request, its dependencies, budget usage and result.
// Pattern: Run use case contracts.
// Usage: Session composition supplies Runner dependencies before Run.
// Related: internal/agent/run/run.go, internal/agent/run/limits.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/policy"
)

type Runner struct {
	PollSteering             func() []string
	Inference                *inference.Settings
	PreviousRun              *runrecord.Checkpoint
	ReasoningEffort          string
	Generator                modelapi.Generator
	Tools                    toolapi.Executor
	Journal                  Journal
	Session, Model, Provider string
	ProjectInstructions      string
	ProjectRoot              string
	Prompts                  *prompt.Templates
	Memory                   *compact.Controller

	Thinking          *bool
	ThinkingSupported bool
	Settings          policy.Settings
	Limits            Limits
	Redact            func(string) string
	RedactionVersion  func() int
	NewStreamMask     func() func(string, bool) string
}
type Result struct {
	End            work.End
	Facts          toolapi.Facts
	Text           string
	Status         runrecord.Status
	HadToolFailure bool
}
type Journal interface {
	ContextMessages(context.Context, string, int64, int) ([]modelapi.Message, error)
	RecordEvent(context.Context, string, runrecord.Event) error
	Commit(context.Context, string, []runrecord.Entry) error
}

type Approver func(context.Context, toolapi.Approval) (bool, error)

var errApprovalDenied = errors.New("operation denied by user; execution stopped")

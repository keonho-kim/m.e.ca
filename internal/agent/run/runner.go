// Purpose: Define one main-agent request, its dependencies, budget usage and result.
// Pattern: Run use case contracts.
// Usage: Session composition supplies Runner dependencies before Run.
// Related: internal/agent/run/run.go, internal/agent/run/limits.go.
package run

import (
	"context"
	"time"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/agent/work"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

type Runner struct {
	Skills                   SkillLoader
	SkillReads               []instruction.Document
	ClientContext            []modelapi.Message
	HideThinking             bool
	ImageInput               bool
	Resume                   bool
	PollSteering             func() []string
	PollContext              func(context.Context) (ContextBatch, error)
	BeforeDecision           func()
	ContextObservations      func([]modelapi.Message) string
	OnAnswer                 func(context.Context, string) (bool, error)
	AdmitModel               func() bool
	Deadline                 time.Time
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

// SkillLoader is the request owner's instruction-read I/O boundary. The
// filesystem reader is supplied by application composition, never constructed here.
type SkillLoader interface {
	Load(context.Context, instruction.Selector, bool) (instruction.Document, error)
}
type Result struct {
	Departed       bool
	End            work.End
	Facts          toolapi.Facts
	Text           string
	Status         runrecord.Status
	HadToolFailure bool
}
type Journal interface {
	ContextMessages(context.Context, string, int64, int, bool, func(int) error) ([]modelapi.Message, error)
	RecordEvent(context.Context, string, runrecord.Event) error
	Commit(context.Context, string, []runrecord.Entry) error
}

type Approver func(context.Context, toolapi.Approval) (bool, error)

var errApprovalDenied = faults.Errorf(diagcatalog.RunOperationDeniedByUserExecutionStopped)

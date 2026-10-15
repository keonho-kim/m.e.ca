// Purpose: Define one serial execution worker dependencies, budget usage and result.
// Pattern: Run use case contracts.
// Usage: Session composition supplies Runner dependencies before Run, DraftPlan or ExecutePlan.
// Related: internal/agent/run/run.go, internal/agent/run/limits.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	planref "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"time"
)

// Usage is owned by one serial worker and shared across approved Plan steps.
type Usage struct {
	Iterations, ModelCalls int
	deadline               time.Time
}
type Runner struct {
	Inference                *inference.Settings
	suppressStream           bool
	observer                 *monitor.Observer
	MonitorSeed              []monitor.Record
	PreviousRun              *Checkpoint
	ReasoningEffort          string
	Generator                modelapi.Generator
	Tools                    toolapi.Executor
	Journal                  Journal
	Session, Model, Provider string
	ProjectInstructions      string
	Prompts                  *prompt.Templates
	Memory                   *compact.Controller
	Step                     *planref.StepRef
	Thinking                 *bool
	ThinkingSupported        bool
	Settings                 policy.Settings
	Limits                   Limits
	Redact                   func(string) string
	MaskStream               func(string) string
	Usage                    *Usage
}
type Result struct {
	Text           string
	Status         Status
	Evidence       string
	HadToolFailure bool
}
type Approver func(context.Context, toolapi.Approval) (bool, error)

var errApprovalDenied = errors.New("operation denied by user; execution stopped")

// Purpose: Define one main-agent request, its dependencies, budget usage and result.
// Pattern: Run use case contracts.
// Usage: Session composition supplies Runner dependencies before Run.
// Related: internal/agent/run/run.go, internal/agent/run/limits.go.
package run

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/compact"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/monitor"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/agent/work"
	"github.com/keonho-kim/v-code/internal/policy"
)

type Runner struct {
	Inference                *inference.Settings
	observer                 *monitor.Observer
	MonitorSeed              []monitor.Record
	PreviousRun              *Checkpoint
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
	MaskStream        func(string) string
}
type Result struct {
	End            work.End
	Facts          toolapi.Facts
	Text           string
	Status         Status
	HadToolFailure bool
}
type Approver func(context.Context, toolapi.Approval) (bool, error)

var errApprovalDenied = errors.New("operation denied by user; execution stopped")

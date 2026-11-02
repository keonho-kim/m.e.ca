// Purpose: Compose identical agent requests from configured settings and a selected session/model.
// Pattern: Application factory with explicit consumer-owned I/O contracts.
// Usage: TUI task submission and cmd/meca headless submission call New.
// Related: internal/agent/session/request.go, internal/tui/session/tasks.go, cmd/meca/headless.go.
package agentrequest

import (
	"context"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/agent/run"
	agentsession "github.com/keonho-kim/meca/internal/agent/session"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
)

type Tools interface {
	toolapi.Executor
	ProjectInstructions(context.Context) (string, error)
}

type Options struct {
	Config       config.Config
	Selection    connection.Selection
	Session      sessionstate.Session
	Generator    modelapi.Generator
	Tools        Tools
	Store        agentsession.Store
	Vault        *credentials.Vault
	LoadPrompts  func() (prompt.Templates, error)
	PollSteering func() []string
	Text         string
}

func New(options Options) agentsession.Request {
	selected := options.Selection
	window := options.Config.ContextLength
	if selected.Details != nil && selected.Details.ContextWindow > 0 {
		window = connection.EffectiveContextLength(window, selected.Details.ContextWindow)
	}
	parameters := options.Config.InferenceSettings()
	var thinking *bool
	if selected.Thinking != nil {
		value := *selected.Thinking
		thinking = &value
	}
	runner := run.Runner{
		ImageInput: selected.Details != nil && selected.Details.ImageInput.Supported(),
		Generator:  options.Generator, Tools: options.Tools, Journal: options.Store,
		Session: options.Session.ID, ProjectRoot: options.Session.ProjectRoot,
		Model: selected.Model, Provider: selected.Connection.ID,
		Thinking: thinking, ThinkingSupported: selected.ThinkingSupported, ReasoningEffort: selected.Effort(),
		Settings: policy.Settings{Scope: policy.System, Permission: options.Config.Permission}, Limits: options.Config.Loop, Inference: &parameters,
		Redact: options.Vault.Redact, RedactionVersion: options.Vault.RedactionVersion,
		NewStreamMask: options.Vault.NewStreamMask, PollSteering: options.PollSteering,
	}
	return agentsession.Request{
		Runner: runner, Store: options.Store, Text: options.Text, ContextWindow: window,
		Compaction: options.Config.Compaction.Settings, LoadPrompts: options.LoadPrompts,
		ProjectInstructions: options.Tools.ProjectInstructions,
	}
}

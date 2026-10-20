// Purpose: Compose saved connections, session resources and noninteractive request presentation.
// Pattern: Composition Root.
// Usage: v [path] --print prompt; v --mode json prompt; v --session ID --export.
// Related: internal/commands/headless/run.go, internal/provider/setup/setup.go.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	agentsession "github.com/keonho-kim/v-code/internal/agent/session"
	"github.com/keonho-kim/v-code/internal/commands/headless"
	"github.com/keonho-kim/v-code/internal/commands/startup"
	"github.com/keonho-kim/v-code/internal/config"
	"github.com/keonho-kim/v-code/internal/config/credentials"
	"github.com/keonho-kim/v-code/internal/config/prompts"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/diagnostic"
	"github.com/keonho-kim/v-code/internal/provider/setup"
	"github.com/keonho-kim/v-code/internal/storage/recovery"
	"github.com/keonho-kim/v-code/internal/storage/sqlite"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"golang.org/x/term"
	"os"
	"strings"
)

func runHeadless(ctx context.Context, mode startup.Mode, cfg config.Config, home, state string, sessions *sqlite.Sessions) (err error) {
	text := mode.Prompt
	if !mode.Export && !mode.Diagnose {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			body, readErr := readPrompt(ctx, os.Stdin)
			if readErr != nil {
				return readErr
			}
			if len(body) > 0 {
				text = string(body) + "\n\n" + text
			}
		}
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("noninteractive mode requires a prompt or stdin")
		}

	}
	var store *sqlite.Store
	if mode.Session != "" {
		store, err = sessions.Open(ctx, mode.Project, mode.Session)
	} else {
		store, err = sessions.Create(ctx, mode.Project, mode.Display)
	}
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, store.Close()) }()
	lease, err := store.LeaseSession()
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, lease.Close()) }()
	id := store.Session().ID
	if mode.Export {
		return store.Export(ctx, id, os.Stdout)
	}
	if err = recovery.Restore(ctx, store, id); err != nil {
		return err
	}
	selected, err := store.Selection(ctx, id)
	if err != nil {
		return err
	}
	if selected.Model == "" && cfg.Model != nil {
		selected, err = cfg.Model.Selection()
		if err != nil {
			return err
		}
	}
	if selected.Model == "" {
		return fmt.Errorf("select a model with !model before using noninteractive mode")
	}
	vault := &credentials.Vault{State: state}
	prepared, err := setup.Prepare(ctx, selected.Connection, "", selected.Model, setup.Defaults{ContextLength: cfg.ContextLength, Thinking: cfg.Thinking}, selected, vault)
	if err != nil {
		return err
	}
	defer prepared.Client.Close()
	selected = *prepared.Selection
	if selected.Identity != nil {
		selected.Identity.Redact(vault.Redact)
	}
	if selected.Details != nil {
		selected.Details.Notice = vault.Redact(selected.Details.Notice)
	}
	if err = store.SaveSelection(ctx, id, selected); err != nil {
		return err
	}
	inference := cfg.InferenceSettings()
	if mode.Diagnose {
		_, err := diagnostic.Check(ctx, prepared.Client, selected.Model, selected.ThinkingSupported, selected.Details != nil && selected.Details.Reasoning != nil && selected.Details.Reasoning.Enabled(), &inference)
		if mode.Output == "json" {
			report := struct {
				Kind   string `json:"kind"`
				Passed bool   `json:"passed"`
				Error  string `json:"error,omitempty"`
			}{Kind: "diagnostic", Passed: err == nil}
			if err != nil {
				report.Error = vault.Redact(err.Error())
			}
			err = errors.Join(err, json.NewEncoder(os.Stdout).Encode(report))
		} else if err == nil {
			_, err = fmt.Fprintln(os.Stdout, "Model diagnostic passed")
		}
		return err
	}
	executor, err := local.NewExecutor(mode.Project, home, state, vault.Redact)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, executor.Close()) }()
	window := cfg.ContextLength
	if selected.Details != nil && selected.Details.ContextWindow > 0 {
		window = connection.EffectiveContextLength(window, selected.Details.ContextWindow)
	}
	runner := run.Runner{Generator: prepared.Client, Tools: executor, Journal: store, Session: id, Model: selected.Model, Provider: selected.Connection.ID, ProjectRoot: mode.Project, Thinking: selected.Thinking, ThinkingSupported: selected.ThinkingSupported, ReasoningEffort: selected.Effort(), Settings: policy.Settings{Scope: policy.System}, Limits: cfg.Loop, Inference: &inference, Redact: vault.Redact, MaskStream: vault.Stream}
	request := agentsession.Request{Runner: runner, Store: store, Text: text, Compaction: cfg.Compaction.Settings, ContextWindow: window, ProjectInstructions: executor.ProjectInstructions, LoadPrompts: func() (prompt.Templates, error) { return prompts.Load(state) }}
	return headless.Run(ctx, request, os.Stdout, mode.Output == "json")
}

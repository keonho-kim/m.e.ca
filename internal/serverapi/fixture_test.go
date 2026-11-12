// Purpose: Compose real storage, policy and tools with a deterministic model for server contract tests.
// Pattern: Integration fixture with explicit resource cleanup.
// Usage: Transport tests share this fixture to compare observable behavior.
// Related: internal/serverapi/transports_test.go, internal/commands/server/tasks.go.
package serverapi

import (
	"context"
	"errors"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/serverapi/detail"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/storage/filemutations"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
	"github.com/keonho-kim/meca/internal/tools/local/filesystem"
)

const testToken = "test-token-0123456789-abcdefghijklmnopqrstuvwxyz"

type generateFunc func(context.Context, model.Request, func(model.Event) error) error

func (g generateFunc) Generate(ctx context.Context, r model.Request, emit func(model.Event) error) error {
	return g(ctx, r, emit)
}

type fixture struct {
	app      *server.Service
	api      *API
	project  string
	requests string
	store    *sqlite.Store
	executed atomic.Int32
}

func newFixture(t *testing.T, g model.Generator, approval bool, concurrency ...int) *fixture {
	t.Helper()
	home := t.TempDir()
	project := t.TempDir()
	state := filepath.Join(home, ".meca")
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := serverrequests.Open(t.Context(), filepath.Join(state, "requests"), server.ProjectScope(project))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := ledger.Close(); err != nil {
			t.Error(err)
		}
	})
	mutations, err := filemutations.Open(t.Context(), filepath.Join(state, "file-mutations"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mutations.Close(); err != nil {
			t.Error(err)
		}
	})
	vault := &credentials.Vault{State: state}
	vault.Remember(testToken)
	pages := detail.NewCache()
	t.Cleanup(func() {
		if err := pages.Close(); err != nil {
			t.Error(err)
		}
	})
	f := &fixture{project: project, requests: filepath.Join(state, "requests")}
	conn, err := connection.New(connection.Ollama, "http://127.0.0.1:11434")
	if err != nil {
		t.Fatal(err)
	}
	selected := connection.Selection{Connection: conn, Model: "fixture"}
	if vision, ok := g.(interface{ ImageInputSupported() bool }); ok && vision.ImageInputSupported() {
		selected.Details = &connection.ModelInfo{ID: "fixture", ImageInput: connection.DeclaredImageInput(true, "fixture")}
	}
	cfg := config.Defaults()
	limit := 1
	if len(concurrency) > 0 {
		limit = concurrency[0]
	}
	boundary, err := filesystem.New(project, home, state)
	if err != nil {
		t.Fatal(err)
	}
	skillReader := skills.New(project, home, skills.Options{Redact: vault.Redact, Check: boundary.SkillTarget, RedactRange: vault.RedactRange, Version: vault.RedactionVersion})
	files, err := local.NewExecutor(project, home, state, vault.Redact, local.Options{Skills: skillReader})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = files.Close() })
	f.app, err = server.New(t.Context(), server.Options{
		Files: files.Files, Mutations: mutations, References: sqlite.ProjectReferences{Sessions: sessions, Project: project},
		MutateEntry: func(ctx context.Context, settings policy.Settings, change fileview.EntryMutation) (string, error) {
			return files.Files.MutateEntry(ctx, settings, change, func(string) error { return nil })
		},
		Read: func(ctx context.Context, id string) (*server.Conversation, error) {
			store, err := sessions.OpenRead(ctx, project, id)
			if err != nil {
				return nil, err
			}
			selected, err := store.Selection(ctx, id)
			if err != nil {
				_ = store.Close()
				return nil, err
			}
			return &server.Conversation{Runtime: agentrequest.Runtime{Session: store.Session(), Selection: selected}, History: store, Close: store.Close}, nil
		},
		Discover: func(ctx context.Context, _ connection.Selection, _ string, _ setup.Defaults) ([]connection.ModelInfo, *connection.Selection, error) {
			return []connection.ModelInfo{{ID: "fixture"}}, nil, ctx.Err()
		},
		Concurrency: limit,
		Config:      cfg, Ledger: ledger, Skills: skillReader,
		List: func(ctx context.Context, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
			return sessions.ConversationPage(ctx, project, cursor, limit)
		},
		Open: func(ctx context.Context, id string, create bool, defaults config.Config) (*server.Conversation, error) {
			var store *sqlite.Store
			var err error
			if create {
				store, err = sessions.Create(ctx, project, project)
			} else {
				store, err = sessions.Open(ctx, project, id)
			}
			if err != nil {
				return nil, err
			}
			lease, err := store.LeaseSession()
			if err != nil {
				_ = store.Close()
				return nil, err
			}
			executor, err := local.NewExecutor(project, home, state, vault.Redact, local.Options{Skills: skillReader, RedactRange: vault.RedactRange, RedactionVersion: vault.RedactionVersion, NewChunkMask: vault.NewChunkMask})
			if err != nil {
				_ = lease.Close()
				_ = store.Close()
				return nil, err
			}
			if err := store.SaveSelection(ctx, store.Session().ID, selected); err != nil {
				_ = executor.Close()
				_ = store.Close()
				_ = lease.Close()
				return nil, err
			}
			f.store = store
			tools := &countTools{Executor: executor, count: &f.executed, approval: approval}
			return &server.Conversation{Team: fixtureTeam(store, home, state, vault), Runtime: agentrequest.Runtime{Config: defaults, Selection: selected, Session: store.Session(), Tools: tools, Skills: skillReader, Store: store, Vault: vault, LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil }}, History: store, Models: store, Close: func() error {
				return errors.Join(pages.CloseSession(store.Session().ID), executor.Close(), store.Close(), lease.Close())
			}, Model: conversationruntime.NewModel(g, nil),
			}, nil
		},
		PrepareModel: func(_ context.Context, _ connection.Selection, _ string, _ setup.Defaults) (server.PreparedModel, error) {
			return server.PreparedModel{Selection: selected, Generator: g}, nil
		},
		UpdateConfig: func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
			apply(&cfg)
			return cfg, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.app.Close(); err != nil {
			t.Error(err)
		}
	})
	f.api = &API{Service: f.app, Remember: vault.Remember, Redact: vault.Redact, RedactRange: vault.RedactRange, NewMask: vault.NewChunkMask, RedactPreview: func(text string) string { return vault.NewStreamMask()(text, true) }, RedactionVersion: vault.RedactionVersion, Details: pages}
	return f
}

type countTools struct {
	*local.Executor
	count    *atomic.Int32
	approval bool
}

func (e *countTools) Prepare(ctx context.Context, p policy.Settings, c tool.Call) (tool.Execution, error) {
	op, err := e.Executor.Prepare(ctx, p, c)
	if err != nil {
		return nil, err
	}
	return countExecution{Execution: op, count: e.count, approval: e.approval}, nil
}

type countExecution struct {
	tool.Execution
	count    *atomic.Int32
	approval bool
}

func (e countExecution) Approval() tool.Approval {
	a := e.Execution.Approval()
	if e.approval {
		a.Required = true
		a.UserRequired = true
	}
	return a
}
func (e countExecution) Execute(ctx context.Context, allow bool) (tool.Result, error) {
	e.count.Add(1)
	return e.Execution.Execute(ctx, allow)
}
func (e countExecution) Release() { tool.Release(e.Execution) }

func deadline(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func waitFinished(t *testing.T, ctx context.Context, app *server.Service, cursor string) server.Task {
	t.Helper()
	for {
		batch, err := app.Next(ctx, cursor)
		if err != nil {
			t.Fatal(err)
		}
		if batch.Snapshot != nil {
			cursor = batch.Snapshot.Cursor
			if batch.Snapshot.Task != nil && batch.Snapshot.Task.Status != "running" && batch.Snapshot.Task.Status != "canceling" {
				return *batch.Snapshot.Task
			}
		}
		for _, event := range batch.Events {
			cursor = event.Cursor
			if event.Record.Kind == "request_finished" {
				state := app.State()
				return *state.Task
			}
		}
	}
}

// Purpose: Verify server request ownership also drives the existing team runtime and attributed events.
// Pattern: Composition integration test using the existing scripted team model.
// Usage: go test -race ./cmd/meca -run TestServerTeamRequest.
// Related: cmd/meca/server_session.go, cmd/meca/team_test.go, internal/commands/server/tasks.go.
package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

func TestServerTeamRequest(t *testing.T) {
	testServerTeamRequest(t, false)
}

func TestServerCompatibleTeamRequest(t *testing.T) {
	testServerTeamRequest(t, true)
}

func testServerTeamRequest(t *testing.T, compatible bool) {
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	home, project := t.TempDir(), t.TempDir()
	cfg, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	state := filepath.Join(home, ".meca")
	if compatible {
		cfg.Permission = policy.PermissionAllMighty
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	ledger, err := serverrequests.Open(ctx, filepath.Join(state, "server-requests"), server.ProjectScope(project))
	if err != nil {
		t.Fatal(err)
	}
	defer ledger.Close()
	vault := &credentials.Vault{State: state}
	model := &compatibleTeamModel{inner: &teamModel{steps: map[string]int{}}}
	conn, err := connection.New(connection.Ollama, "http://127.0.0.1:11434")
	if err != nil {
		t.Fatal(err)
	}
	app, err := server.New(ctx, server.Options{Config: cfg, Ledger: ledger, Responses: ledger, Project: project,
		ReadOutput: func(ctx context.Context, id string, after, through int64, maxBytes int, reserve func(int) error) (string, error) {
			return sessions.ResponseOutput(ctx, project, id, after, through, maxBytes, reserve)
		},
		Open: func(ctx context.Context, id string, create bool, defaults config.Config) (*server.Conversation, error) {
			c, err := openServerSession(ctx, startup.Mode{Project: project, Display: project}, defaults, home, state, sessions, vault, nil, id, create)
			if err != nil {
				return nil, err
			}
			c.Model = conversationruntime.NewModel(model, nil)
			c.Runtime.Selection = connection.Selection{Connection: conn, Model: "test-model"}
			return c, nil
		},
		List: func(ctx context.Context, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
			return sessions.ConversationPage(ctx, project, cursor, limit)
		},
		PrepareModel: func(context.Context, connection.Selection, string, setup.Defaults) (server.PreparedModel, error) {
			return server.PreparedModel{}, fmt.Errorf("unexpected model setup")
		},
		UpdateConfig: func(ctx context.Context, apply func(*config.Config)) (config.Config, error) {
			return config.Update(ctx, state, apply)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := app.Close(); err != nil {
			t.Error(err)
		}
	}()
	selected, err := app.Select(ctx, "", true)
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	if _, err := app.Configure(ctx, nil, &enabled); err != nil {
		t.Fatal(err)
	}
	var task server.Task
	var submission server.Submitted
	cursor := selected.Cursor
	if compatible {
		submission, err = app.Submit(ctx, server.Submission{Kind: "response", Stored: true, Messages: []server.InputMessage{{Role: "developer", Text: "client context marker"}, {Role: "user", Text: "Create independent contributions inside the project."}}})
		task.ID, cursor = submission.Response.Task, submission.Cursor
	} else {
		task, err = app.Start(ctx, selected.Session.ID, "server-team-request", "Create independent contributions inside the project.")
	}
	if err != nil {
		t.Fatal(err)
	}
	agents := map[string]bool{}
	finished := false
	finalText := ""
	for !finished {
		batch, err := app.Next(ctx, cursor)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range batch.Events {
			cursor = e.Cursor
			if e.Record.Final {
				finalText = e.Record.Text
			}
			if e.Record.Agent != "" {
				agents[e.Record.Agent] = true
			}
			if e.Record.Kind == "approval_required" {
				if err := app.Approve(task.ID, e.ApprovalID, true); err != nil {
					t.Fatal(err)
				}
			}
			finished = finished || e.Record.Kind == "request_finished"
		}
	}
	status, err := app.TaskStatus(ctx, task.ID)
	if err != nil || status.Status != "answered" || len(agents) < 2 {
		t.Fatal(status, agents, err)
	}
	if compatible {
		value, err := app.ResponseState(ctx, submission.Response.ID, true)
		if err != nil || value.Status != "completed" {
			t.Fatal(value, err)
		}
		text, err := app.ResponseResult(ctx, value, nil)
		if err != nil || text != strings.TrimSpace(finalText) || !model.sawContext.Load() {
			t.Fatal(text, finalText, model.sawContext.Load(), err)
		}
	}
}

type compatibleTeamModel struct {
	inner      *teamModel
	sawContext atomic.Bool
}

func (g *compatibleTeamModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	for _, message := range request.Messages {
		if strings.Contains(message.Content, "client context marker") {
			g.sawContext.Store(true)
		}
	}
	return g.inner.Generate(ctx, request, emit)
}

//go:build !tui

// Purpose: Exercise real engine, native provider, tools, journal and SSE rendering as one PGO workload.
// Pattern: Isolated end-to-end benchmark with deterministic loopback model responses.
// Usage: BenchmarkEngineWorkload supplies synthetic CPU profiles; TestEngineWorkload validates every dialect.
// Related: cmd/meca/engine_provider_test.go, cmd/meca/engine_stream_test.go, scripts/performance/pgo.sh.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	conversationruntime "github.com/keonho-kim/meca/internal/commands/conversation"

	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/provider/setup"
	"github.com/keonho-kim/meca/internal/serverapi"
	"github.com/keonho-kim/meca/internal/storage/serverrequests"
	sessionstate "github.com/keonho-kim/meca/internal/storage/session"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

const engineToolRounds = 6

func BenchmarkEngineWorkload(b *testing.B) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.Claude, connection.Gemini} {
		b.Run(string(kind), func(b *testing.B) {
			run := newEngineWorkload(b, b.Context(), kind)
			b.ReportAllocs()
			for b.Loop() {
				if err := run(); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestEngineWorkload(t *testing.T) {
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.Claude, connection.Gemini} {
		t.Run(string(kind), func(t *testing.T) {
			run := newEngineWorkload(t, t.Context(), kind)
			for range 2 {
				if err := run(); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func newEngineWorkload(tb testing.TB, ctx context.Context, kind connection.ProviderKind) func() error {
	tb.Helper()
	home, project := tb.TempDir(), tb.TempDir()
	cfg, err := bootstrap.Initialize(home)
	if err != nil {
		tb.Fatal(err)
	}
	cfg.Permission, cfg.ContextLength = policy.PermissionAllMighty, 262144
	state := filepath.Join(home, ".meca")
	if err := os.WriteFile(filepath.Join(project, "fixture.txt"), []byte(strings.Repeat("Immutable source observation.\n", 2048)), 0600); err != nil {
		tb.Fatal(err)
	}
	var calls atomic.Uint64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		engineProvider(w, r, kind, int(calls.Add(1)-1)%(engineToolRounds+1))
	}))
	tb.Cleanup(provider.Close)
	conn, err := connection.New(kind, provider.URL)
	if err != nil {
		tb.Fatal(err)
	}
	client, err := httpclient.NewClient(conn, "benchmark-only-key")
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(client.Close)
	if err := client.SetContextLength(cfg.ContextLength); err != nil {
		tb.Fatal(err)
	}
	selected := connection.Selection{Connection: conn, Model: "fixture", Details: &connection.ModelInfo{ID: "fixture", ContextWindow: cfg.ContextLength}}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		tb.Fatal(err)
	}
	ledger, err := serverrequests.Open(ctx, filepath.Join(state, "requests"), server.ProjectScope(project))
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := ledger.Close(); err != nil {
			tb.Error(err)
		}
	})
	vault := &credentials.Vault{State: state}
	if err := vault.Remember("benchmark-only-key"); err != nil {
		tb.Fatal(err)
	}
	app, err := server.New(ctx, server.Options{
		Config: cfg, Ledger: ledger,
		Open: func(ctx context.Context, id string, create bool, defaults config.Config) (*server.Conversation, error) {
			c, err := openServerSession(ctx, startup.Mode{Project: project, Display: project}, defaults, home, state, sessions, vault, nil, id, create)
			if err == nil {
				c.InheritsModel = false
				c.Runtime.Selection = selected
				c.Model = conversationruntime.NewModel(client, nil)
			}
			return c, err
		},
		List: func(ctx context.Context, cursor string, limit int) ([]sessionstate.Conversation, string, error) {
			return sessions.ConversationPage(ctx, project, cursor, limit)
		},
		PrepareModel: func(context.Context, connection.Selection, string, setup.Defaults) (server.PreparedModel, error) {
			return server.PreparedModel{Selection: selected, Generator: client}, nil
		},
		UpdateConfig: func(_ context.Context, apply func(*config.Config)) (config.Config, error) {
			apply(&cfg)
			return cfg, nil
		},
	})
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		if err := app.Close(); err != nil {
			tb.Error(err)
		}
	})
	serveCtx, stopServe := context.WithCancel(ctx)
	ready, done := make(chan string, 1), make(chan error, 1)
	const token = "native-engine-workload-token-0123456789"
	api := &serverapi.API{Service: app, Redact: vault.Redact, NewMask: vault.NewChunkMask, RedactPreview: func(text string) string { return vault.NewStreamMask()(text, true) }, RedactionVersion: vault.RedactionVersion}
	go func() {
		done <- serverapi.ListenAndServe(serveCtx, api, token, serverapi.Options{Protocol: "sse", Listen: "127.0.0.1:0"}, func(address string) { ready <- address })
	}()
	var address string
	select {
	case address = <-ready:
	case err := <-done:
		stopServe()
		tb.Fatal(err)
	case <-ctx.Done():
		stopServe()
		<-done
		tb.Fatal(ctx.Err())
	}
	tb.Cleanup(func() {
		stopServe()
		if err := <-done; err != nil && !errors.Is(err, context.Canceled) {
			tb.Error(err)
		}
	})
	sequence := 0
	return func() error {
		operation, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		before := calls.Load()
		selected, err := app.Select(operation, "", true)
		if err != nil {
			return err
		}
		sequence++
		text := strings.Repeat("Immutable request context.\n", 2048) + "Read fixture.txt for each requested observation, then return the final report."
		task, err := app.Start(operation, selected.Session.ID, fmt.Sprintf("workload-%d", sequence), text)
		if err != nil {
			return err
		}
		observer, stopObserver := context.WithCancel(operation)
		defer stopObserver()
		request, err := http.NewRequestWithContext(observer, http.MethodGet, address+"/v1/events?cursor="+selected.Cursor, nil)
		if err != nil {
			return err
		}
		request.Header.Set("Authorization", "Bearer "+token)
		reply, err := http.DefaultClient.Do(request)
		if err != nil {
			return err
		}
		defer reply.Body.Close()
		output := engineStream{header: make(http.Header), status: http.StatusOK}
		buffer := make([]byte, 64<<10)
		for !output.completed && output.err == nil {
			n, err := reply.Body.Read(buffer)
			_, _ = output.Write(buffer[:n])
			if err != nil && (!errors.Is(err, io.EOF) || !output.completed) {
				return err
			}
		}
		stopObserver()
		if output.status != http.StatusOK || !output.completed || output.err != nil {
			return fmt.Errorf("engine workload failed: status=%d completed=%v: %w", output.status, output.completed, errors.Join(output.err, fmt.Errorf("%s", output.failure)))
		}
		if status, err := app.TaskStatus(operation, task.ID); err != nil || status.Status != "answered" {
			return fmt.Errorf("native workload did not finish: %s %w", status.Status, err)
		}
		if got := calls.Load() - before; got != engineToolRounds+1 {
			return fmt.Errorf("engine workload made %d model calls, want %d", got, engineToolRounds+1)
		}
		return nil
	}
}

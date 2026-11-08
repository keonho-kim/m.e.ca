// Purpose: Exercise real engine, native provider, tools, journal and SSE rendering as one PGO workload.
// Pattern: Isolated end-to-end benchmark with deterministic loopback model responses.
// Usage: BenchmarkEngineWorkload supplies synthetic CPU profiles; TestEngineWorkload validates every dialect.
// Related: cmd/meca/engine_provider_test.go, cmd/meca/engine_stream_test.go, scripts/performance/pgo.sh.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/keonho-kim/meca/internal/serverapi/openai"
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
		Project: project, Config: cfg, Ledger: ledger, Responses: ledger,
		ReadOutput: func(ctx context.Context, id string, after, through int64, max int) (string, error) {
			return sessions.ResponseOutput(ctx, project, id, after, through, max)
		},
		Open: func(ctx context.Context, id string, create bool, defaults config.Config) (*server.Conversation, error) {
			c, err := openServerSession(ctx, startup.Mode{Project: project, Display: project}, defaults, home, state, sessions, vault, nil, id, create)
			if err == nil {
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
	mux := http.NewServeMux()
	(&openai.Handler{Service: app, Redact: vault.Redact, Streams: make(chan struct{}, 16)}).Mount(mux)
	body, err := json.Marshal(map[string]any{"model": "meca", "stream": true, "store": true, "input": []map[string]string{
		{"role": "developer", "content": strings.Repeat("Immutable request context.\n", 2048)},
		{"role": "user", "content": "Read fixture.txt for each requested observation, then return the final report."},
	}})
	if err != nil {
		tb.Fatal(err)
	}
	return func() error {
		operation, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		before := calls.Load()
		request := httptest.NewRequestWithContext(operation, http.MethodPost, "http://meca.local/v1/responses", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		output := engineStream{header: make(http.Header), status: http.StatusOK}
		mux.ServeHTTP(&output, request)
		if output.status != http.StatusOK || !output.completed || output.err != nil {
			return fmt.Errorf("engine workload failed: status=%d completed=%v: %w", output.status, output.completed, errors.Join(output.err, fmt.Errorf("%s", output.failure)))
		}
		if _, err := app.WaitResponse(ctx, output.header.Get("X-Request-ID")); err != nil {
			return err
		}
		if got := calls.Load() - before; got != engineToolRounds+1 {
			return fmt.Errorf("engine workload made %d model calls, want %d", got, engineToolRounds+1)
		}
		return nil
	}
}

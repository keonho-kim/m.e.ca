// Purpose: Serve real embedded UI and project runtimes against a credential-free local model fixture.
// Pattern: Opt-in browser integration fixture.
// Usage: MECA_UI_DESCRIPTOR=/tmp/task/connection.json go test ./cmd/meca -run TestUIBrowserFixture -v.
// Related: cmd/meca/ui_provider_test.go, webui/e2e/workspace.spec.ts.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/startup"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/bootstrap"
	"github.com/keonho-kim/meca/internal/diagnostics"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
)

type uiFixtureLog struct {
	mu    sync.Mutex
	ready chan string
}

func (w *uiFixtureLog) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var entry struct {
		Message string `json:"msg"`
		Listen  string `json:"listen"`
	}
	if json.Unmarshal(p, &entry) == nil && entry.Message == "server_listening" {
		select {
		case w.ready <- entry.Listen:
		default:
		}
	}
	return len(p), nil
}

func TestUIBrowserFixture(t *testing.T) {
	descriptor := os.Getenv("MECA_UI_DESCRIPTOR")
	if descriptor == "" {
		t.Skip("set MECA_UI_DESCRIPTOR to run the browser integration fixture")
	}
	home := t.TempDir()
	state := filepath.Join(home, ".meca")
	projectA, projectB := filepath.Join(home, "alpha"), filepath.Join(home, "beta")
	for _, project := range []string{projectA, projectB} {
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "sample.go"), []byte("package sample\n\nfunc Value() int { return 1 }\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("# Browser fixture\n\nThis is isolated integration test content.\n"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "fixture.png"), encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := bootstrap.Initialize(home)
	if err != nil {
		t.Fatal(err)
	}
	provider := uiProvider(t, home)
	cfg.Model = &config.SavedModel{Provider: connection.VLLM, Endpoint: provider.URL, Name: "fixture-model"}
	cfg.Workspaces = []string{projectB}
	if err := config.Save(state, cfg); err != nil {
		t.Fatal(err)
	}
	sessions, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	archived, err := sessions.Create(t.Context(), projectA, projectA)
	if err != nil {
		t.Fatal(err)
	}
	archiveID := archived.Session().ID
	records := make([]record.Entry, 300)
	for index := range records {
		records[index] = record.Entry{Message: &model.Message{Role: "assistant", Content: fmt.Sprintf("Archive message %03d", index)}}
	}
	if err := archived.Commit(t.Context(), archiveID, records); err != nil {
		t.Fatal(err)
	}
	if err := archived.Close(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	log := &uiFixtureLog{ready: make(chan string, 1)}
	sink, err := diagnostics.New(diagnostics.Options{Writer: log, Level: slog.LevelDebug, Redact: func(s string) string { return s }, Interrupt: func() error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	ctx = diagnostics.WithSink(ctx, sink)
	mode := startup.Mode{Project: projectA, Display: projectA, Server: true, ServerOptions: startup.ServerOptions{UI: true, Listen: "127.0.0.1:0", Protocol: "sse", Concurrency: 4}}
	done := make(chan error, 1)
	go func() { done <- runServer(ctx, mode, cfg, home, state, sessions) }()
	var address string
	select {
	case address = <-log.ready:
	case err := <-done:
		t.Fatal(err)
	case <-time.After(15 * time.Second):
		t.Fatal("UI listener did not become ready")
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatal(err)
	}
	defer watcher.Close()
	if err := watcher.Add(filepath.Dir(descriptor)); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(map[string]string{"url": address, "tokenFile": filepath.Join(state, "server.token"), "projectA": projectA, "projectB": projectB, "home": home, "provider": provider.URL, "archiveSession": archiveID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptor, data, 0600); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(descriptor)
	stop := descriptor + ".stop"
	deadline := time.NewTimer(45 * time.Minute)
	defer deadline.Stop()
	for {
		select {
		case <-watcher.Events:
			if _, err := os.Stat(stop); err == nil {
				cancel()
				_ = os.Remove(stop)
				if err := <-done; err != nil {
					t.Fatal(err)
				}
				return
			}
		case err := <-watcher.Errors:
			t.Fatal(err)
		case err := <-done:
			t.Fatal("UI stopped before the browser completed:", err)
		case <-deadline.C:
			cancel()
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			return
		case <-ctx.Done():
			return
		}
	}
}

var _ io.Writer = (*uiFixtureLog)(nil)

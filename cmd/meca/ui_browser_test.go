//go:build !tui

// Purpose: Serve real embedded UI and project runtimes against a credential-free local model fixture.
// Pattern: Test-owned browser integration server.
// Usage: Playwright starts one instance per test and stops it with SIGTERM.
// Related: cmd/meca/ui_provider_test.go, webui/e2e/browser-server.ts.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

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
	cfg, err := bootstrap.Initialize(t.Context(), home)
	if err != nil {
		t.Fatal(err)
	}
	provider := uiProvider(t, home)
	cfg.Language = "ko"
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
	ctx, stop := signal.NotifyContext(t.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log := &uiFixtureLog{ready: make(chan string, 1)}
	sink, err := diagnostics.New(diagnostics.Options{Writer: log, Level: slog.LevelDebug, Redact: func(s string) string { return s }, Interrupt: func() error { return nil }})
	if err != nil {
		t.Fatal(err)
	}
	defer sink.Close()
	ctx = diagnostics.WithSink(ctx, sink)
	mode := startup.Mode{Project: projectA, Display: projectA, Server: true, ServerOptions: startup.ServerOptions{UI: true, Listen: "127.0.0.1:0", Protocol: "sse", Concurrency: 4}}
	mode.ServerOptions.Certificate = os.Getenv("MECA_UI_CERTIFICATE")
	mode.ServerOptions.Key = os.Getenv("MECA_UI_KEY")
	done := make(chan struct{})
	var serverErr error
	go func() {
		defer close(done)
		serverErr = runServer(ctx, mode, cfg, home, state, sessions, nil)
	}()
	defer func() {
		stop()
		<-done
		if serverErr != nil {
			t.Error(serverErr)
		}
	}()
	var address string
	select {
	case address = <-log.ready:
	case <-done:
		t.Fatal("UI server stopped before publishing its address")
	case <-ctx.Done():
		return
	case <-time.After(15 * time.Second):
		t.Fatal("UI listener did not become ready")
	}
	data, err := json.Marshal(map[string]string{"url": address, "tokenFile": filepath.Join(state, "server.token"), "projectA": projectA, "projectB": projectB, "home": home, "provider": provider.URL, "archiveSession": archiveID})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(descriptor+".tmp", data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(descriptor+".tmp", descriptor); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ctx.Done():
	case <-done:
		if ctx.Err() == nil {
			t.Fatal("UI server stopped before browser teardown")
		}
	}
}

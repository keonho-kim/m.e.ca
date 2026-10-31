// Purpose: Exercise background LAT memory with live work, steering and independent stored sources.
// Pattern: Scripted-model integration over real local files and SQLite.
// Usage: go test -race ./internal/agent/session -run TestLATBackgroundMemory.
// Related: internal/agent/run/lat_memory.go, internal/storage/sqlite/lat_sources.go.
package session_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/session"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/commands/agentrequest"
	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/config/credentials"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestLATBackgroundMemoryRetainsSteeringAndConcurrentEffects(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	project, home, state := t.TempDir(), t.TempDir(), t.TempDir()
	catalog, err := sqlite.OpenSessions(filepath.Join(state, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := catalog.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	vault := &credentials.Vault{State: state}
	tools, err := local.NewExecutor(project, home, state, vault.Redact)
	if err != nil {
		t.Fatal(err)
	}
	defer tools.Close()
	if err = store.AppendMessage(ctx, store.Session().ID, modelapi.Message{Role: "user", Content: "EARLY_REQUIRED_CONSTRAINT\n" + strings.Repeat("Keep a focused API.\n", 5000)}); err != nil {
		t.Fatal(err)
	}
	if err = store.InstallMemory(ctx, store.Session().ID, 0, compact.Memory{Revision: 1, Through: 1, Markdown: "POISON_MAIN_SUMMARY"}); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var released sync.Once
	t.Cleanup(func() { released.Do(func() { close(release) }) })
	var pending atomic.Bool
	var summaries, reviews atomic.Int32
	steps := 0
	g := generation(func(ctx context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		usage := &modelapi.TokenUsage{Input: 100, Output: 5, Cost: new(0.001)}
		if strings.HasPrefix(request.Messages[0].Content, "# Preserve requirements and observed work for review") {
			summaries.Add(1)
			if strings.Contains(request.Messages[1].Content, "POISON_MAIN_SUMMARY") || !strings.Contains(request.Messages[1].Content, "EARLY_REQUIRED_CONSTRAINT") {
				t.Error("review memory did not use independent originals")
			}
			close(started)
			select {
			case <-release:
			case <-ctx.Done():
				return ctx.Err()
			}
			return send(modelapi.Event{Delta: "- EARLY_REQUIRED_CONSTRAINT: keep a focused API.\n- a.txt was actually written.", Usage: usage, Done: true})
		}
		if len(request.Tools) == 0 {
			reviews.Add(1)
			all := request.Messages[1].Content
			for _, value := range []string{"EARLY_REQUIRED_CONSTRAINT", "LATEST_USER_CONSTRAINT", "b.txt", "c.txt"} {
				if !strings.Contains(all, value) {
					t.Error("final review lost", value)
				}
			}
			if strings.Contains(all, "POISON_MAIN_SUMMARY") || strings.Contains(all, "PRIVATE_MAIN_THINKING") {
				t.Error("main summary or thinking leaked to independent review")
			}
			return send(modelapi.Event{Delta: "NO_FINDINGS", Usage: usage, Done: true})
		}
		steps++
		if steps == 2 {
			pending.Store(true)
		}
		if steps == 3 {
			select {
			case <-started:
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		if steps == 4 {
			return send(modelapi.Event{Delta: "작업을 마쳤습니다.", Usage: usage, Done: true})
		}
		path := []string{"a.txt", "b.txt", "c.txt"}[steps-1]
		args, _ := json.Marshal(map[string]string{"path": path, "content": "actual content\n"})
		return send(modelapi.Event{Thinking: "PRIVATE_MAIN_THINKING", Calls: []toolapi.Call{{ID: path, Name: "write_file", Arguments: args}}, Usage: usage, Done: true})
	})
	request := agentrequest.New(agentrequest.Options{LAT: true, Config: config.Defaults(), Selection: connection.Selection{Model: "selected", Connection: connection.Connection{ID: "provider"}}, Session: store.Session(), Generator: g, Tools: tools, Store: store, Vault: vault, Text: "파일 세 개를 작성해 주세요", LoadPrompts: func() (prompt.Templates, error) { return prompt.Defaults(), nil }, PollSteering: func() []string {
		if pending.Swap(false) {
			return []string{"LATEST_USER_CONSTRAINT: 기존 요청을 유지하고 마지막 파일도 작성하세요"}
		}
		return nil
	}})
	result, err := session.Execute(ctx, request, func(context.Context, toolapi.Approval) (bool, error) { return true, nil }, func(event runrecord.Event) error {
		if event.Kind == "tool_succeeded" && event.CallID == "c.txt" {
			released.Do(func() { close(release) })
		}
		return nil
	})
	if err != nil || result.Status != runrecord.Answered || steps != 4 || summaries.Load() != 1 || reviews.Load() != 1 {
		t.Fatalf("result=%+v steps=%d summaries=%d reviews=%d error=%v", result, steps, summaries.Load(), reviews.Load(), err)
	}
	for _, name := range []string{"a.txt", "b.txt", "c.txt"} {
		if _, err = os.Stat(filepath.Join(project, name)); err != nil {
			t.Fatal("foreground work was blocked by compaction", err)
		}
	}
	memo, err := store.LATMemory(ctx, store.Session().ID)
	if err != nil || memo.Revision != 1 || !strings.Contains(memo.Text, "EARLY_REQUIRED_CONSTRAINT") {
		t.Fatal("independent memo not committed", memo, err)
	}
	main, err := store.Memory(ctx, store.Session().ID)
	if err != nil || main.Markdown != "POISON_MAIN_SUMMARY" {
		t.Fatal("LAT modified main compaction memory", err)
	}
	counts, err := store.TokenUsage(ctx, store.Session().ID)
	if err != nil || counts.Input != 600 || counts.Output != 30 {
		t.Fatal("concurrent model usage omitted or doubled", counts, err)
	}
	rows, _, err := store.Timeline(ctx, store.Session().ID, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	var measuredRole bool
	var exported strings.Builder
	if err = store.Export(ctx, store.Session().ID, &exported); err != nil {
		t.Fatal(err)
	}
	measuredRole = strings.Contains(exported.String(), `"role":"`+string(inference.LATCompaction)+`"`)
	if len(rows) == 0 || !measuredRole {
		t.Fatal("background attempt provenance was not recorded")
	}
}

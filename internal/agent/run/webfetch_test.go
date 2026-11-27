// Purpose: Verify WebFetch through native tool execution, SQLite and context recovery.
// Pattern: Scripted model over a real local document tool and journal.
// Usage: go test ./internal/agent/run -run TestWebFetch.
// Related: internal/agent/run/webfetch_budget.go, internal/agent/run/observation_pages.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/modeltest"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/compact"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/storage/sqlite"
	"github.com/keonho-kim/meca/internal/tools/browser/webfetch"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestWebFetchCompactionPreservesPendingPage(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// This fixture serves plain text. Installation discovery does not start Chromium.
	t.Setenv("MECA_BROWSER_EXECUTABLE", executable)
	const pageBytes = 256
	const first, second, third = "FIRST_SECTION", "PENDING_SECTION", "LAST_SECTION"
	var source strings.Builder
	for _, marker := range []string{first, second, third} {
		source.WriteString(marker)
		source.WriteString(strings.Repeat("x", pageBytes-len(marker)))
	}
	source.WriteString("UNREAD_SUFFIX")
	var downloads atomic.Int32
	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downloads.Add(1)
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, source.String())
	}))
	defer httpServer.Close()

	ctx := t.Context()
	project, home := t.TempDir(), t.TempDir()
	sessions, err := sqlite.OpenSessions(filepath.Join(home, "sessions"))
	if err != nil {
		t.Fatal(err)
	}
	db, err := sessions.Create(ctx, project, project)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	id := db.Session().ID
	if err := db.AppendMessage(ctx, id, modelapi.Message{Role: "user", Content: "Previous topic to compact."}); err != nil {
		t.Fatal(err)
	}
	redact := func(text string) string { return text }
	client := webfetch.NewClient()
	tools, err := local.NewExecutor(project, home, filepath.Join(home, ".meca"), redact, local.Options{WebFetch: client})
	if err != nil {
		t.Fatal(err)
	}
	defer tools.Close()
	step, compactions := 0, 0
	pending := ""
	script := boundaryModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		step++
		var result string
		var live []toolapi.ObservationCursor
		for _, message := range request.Messages {
			if message.Role == "tool" && message.ToolName == "codemode" {
				result = message.Content
			}
			if strings.HasPrefix(message.Content, "Unread observations.") {
				_, body, _ := strings.Cut(message.Content, "\n")
				if err := json.Unmarshal([]byte(body), &live); err != nil {
					return err
				}
			}
		}
		call := func(input map[string]any) error {
			input["limit"] = pageBytes
			body, err := json.Marshal(input)
			if err != nil {
				return err
			}
			return emit(modelapi.Event{Done: true, Calls: modeltest.Calls([]toolapi.Call{{Name: toolapi.WebFetch, ID: fmt.Sprintf("call_page_%d", step), Arguments: body}})})
		}
		next := func(offset int) error {
			if len(live) != 1 || live[0].Tool != toolapi.WebFetch || live[0].NextOffset != offset {
				t.Fatalf("step %d, unexpected continuation: %+v", step, live)
			}
			return call(map[string]any{"result_id": live[0].ResultID, "revision": live[0].Revision, "offset": live[0].NextOffset})
		}
		if step >= 3 && containsPage(request.Messages, first) {
			t.Fatal("consumed first page returned to model context")
		}
		switch step {
		case 1:
			if !slices.ContainsFunc(request.Tools, func(d toolapi.Definition) bool {
				return d.Name == "codemode" && strings.Contains(d.Description, toolapi.WebFetch+"(args:")
			}) ||
				!strings.Contains(request.Messages[0].Content, "Use web_fetch") {
				t.Fatal("configured tool or guidance is missing")
			}
			return call(map[string]any{"url": httpServer.URL})
		case 2:
			if !strings.Contains(result, first) || strings.Contains(result, second) {
				t.Fatalf("first model request did not receive exactly the first page: result=%q cursors=%+v downloads=%d", result, live, downloads.Load())
			}
			return next(pageBytes)
		case 3:
			pending = result
			if !strings.Contains(pending, second) {
				t.Fatal("second page did not reach model")
			}
			return &modelapi.ContextLimitError{Input: 24769, RequestedOutput: 8192, Capacity: 32768}
		case 4:
			if len(request.Tools) != 0 || containsPage(request.Messages, second) {
				t.Fatal("pending page leaked into the compaction call")
			}
			return emit(modelapi.Event{Done: true, Delta: "The current task reads an internal document."})
		case 5:
			if result != pending {
				t.Fatal("context recovery lost the unacknowledged page")
			}
			return next(2 * pageBytes)
		case 6:
			if !strings.Contains(result, third) || containsPage(request.Messages, second) || strings.Contains(result, "UNREAD_SUFFIX") {
				t.Fatal("continuation failed to advance exactly one page")
			}
			return emit(modelapi.Event{Done: true, Delta: "done"})
		default:
			t.Fatal("unexpected model call")
			return nil
		}
	})
	inferenceSettings := inference.Defaults()
	inferenceSettings.Agent.MaxOutputTokens = 8192
	r := Runner{
		Generator: script, Tools: tools, Journal: db, Session: id, Model: "fixture", ProjectRoot: project, Redact: redact,
		Inference: &inferenceSettings,
		Settings:  policy.Settings{Scope: policy.System},
		Memory:    &compact.Controller{ContextWindow: 65536, Repository: db, Session: id, Model: "fixture", Prompt: prompt.Defaults().Compaction, Settings: compact.Defaults(), Redact: redact},
	}
	result, err := r.Run(ctx, "Read the internal document at "+httpServer.URL, approve, func(event runrecord.Event) error {
		if event.Kind == "compaction" {
			compactions++
		}
		return nil
	})
	if err != nil || result.Text != "done" || step != 6 || compactions != 1 || downloads.Load() != 1 {
		t.Fatalf("result=%+v calls=%d compactions=%d downloads=%d err=%v", result, step, compactions, downloads.Load(), err)
	}
	if cursors, err := tools.ObservationCursors(ctx); err != nil || len(cursors) != 0 {
		t.Fatalf("finished run retained an unread cursor: %+v %v", cursors, err)
	}
	restored, err := db.ContextMessages(ctx, id, 0, 32<<10, false, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range restored {
		if message.ToolName == toolapi.WebFetch && (message.Content != toolapi.ObservationRead || message.TransientObservation) {
			t.Fatal("SQLite rehydrated an acknowledged page")
		}
	}
}

func TestWebFetchBudgetSharesProviderHeadroom(t *testing.T) {
	s := execution{limits: DefaultLimits()}
	request := modelapi.Request{MaxOutput: 1024}
	answer := modelapi.Response{Calls: []toolapi.Call{{Name: toolapi.WebFetch}, {Name: toolapi.WebFetch}}}
	server := modelapi.InputBudget{Window: 8192, InputTokens: 3700, OutputTokens: 2048}
	measured := modelapi.TokenUsage{Input: 3750, Output: 40}
	got := s.pageAllowance(request, answer, server, measured, 0)
	want := (8192 - 2048 - 3750 - 40 - 3*256) / 2
	if got != want {
		t.Fatalf("per-page budget = %d; want %d", got, want)
	}
	measured.Input = 8150
	if got := s.pageAllowance(request, answer, server, measured, 0); got != 0 {
		t.Fatalf("allocated %d bytes in an exhausted context", got)
	}
}

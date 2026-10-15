// Purpose: Exercise routing and review contracts through real filesystem adapters without live models.
// Pattern: Scripted integration regressions for observed failures.
// Usage: go test ./internal/agent/run -run TestAlignment.
// Related: internal/agent/run/review_context.go, internal/agent/run/batch_wave.go, internal/tools/local/directions.go.
package run

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAlignmentLongReviewReasonReachesReadAndStorage(t *testing.T) {
	r, _, root, j, _ := realBatchRunner(t, nil)
	support := filepath.Join(root, "support.txt")
	if err := os.WriteFile(support, []byte("known value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	reason := strings.Repeat("Inspect this source before approving the proposed change. ", 100) + "\nPreserve the source exactly."
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new\n"}), "Created a."}}
	count := 0
	g.review = func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		count++
		reply := `{"decision":"pass"}`
		if count == 1 {
			body, _ := json.Marshal(map[string]any{"decision": "needs_context", "context_requests": []any{map[string]any{"path": support, "offset": 0, "limit": 4096, "reason": reason}}})
			reply = string(body)
		} else if !strings.Contains(r.Messages[len(r.Messages)-1].Content, "known value") {
			t.Fatal("scoped read missing from review")
		}
		return emit(modelapi.Event{Delta: reply, Done: true})
	}
	r.Generator = g
	result, err := r.Run(context.Background(), "create a using support.txt", approve, emit)
	if err != nil || result.Status != Answered || count != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	found := false
	for _, e := range j.events {
		if e.Kind == "self_grounding" && e.Internal && e.Grounding.Reason == reason {
			found = true
		}
	}
	if !found {
		t.Fatal("long review reason was truncated or not persisted")
	}
}

func TestAlignmentShellCreationFeedbackNamesActualNativeArguments(t *testing.T) {
	r, _, root, j, _ := realBatchRunner(t, nil)
	g := &generator{replies: []string{nativeCall("execute_command", map[string]any{"command": "mkdir -p src/app"}), nativeCall("create_directory", map[string]any{"paths": []string{"src/app"}}), "Created src/app."}}
	r.Generator = g
	result, err := r.Run(context.Background(), "create src/app", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	if info, err := os.Stat(filepath.Join(root, "src/app")); err != nil || !info.IsDir() {
		t.Fatal("native recovery did not create directory")
	}
	for _, m := range j.messages {
		if m.Role == "tool" && m.ToolName == "execute_command" {
			if !strings.Contains(m.Content, "paths: an array") || !strings.Contains(m.Content, "Nothing from the rejected shell command ran") {
				t.Fatalf("unclear routing: %s", m.Content)
			}
			return
		}
	}
	t.Fatal("missing structured routing feedback")
}

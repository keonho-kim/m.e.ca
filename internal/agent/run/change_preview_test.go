// Purpose: Verify completed changes retain their frozen preview without copying it into model tool results.
// Pattern: Runtime event and bounded-payload tests.
// Usage: go test ./internal/agent/run -run TestChangePreview.
// Related: internal/agent/run/execute.go, internal/agent/run/change_preview.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
	"unicode/utf8"
)

type previewExecutor struct {
	executor
	preview string
}

func (e *previewExecutor) Prepare(ctx context.Context, settings policy.Settings, planning bool, c toolapi.Call) (toolapi.Execution, error) {
	_, err := e.executor.Prepare(ctx, settings, planning, c)
	return e, err
}
func (e *previewExecutor) Approval() toolapi.Approval {
	a := e.executor.Approval()
	if len(e.calls) > 0 && e.calls[len(e.calls)-1].Name == "write_file" {
		a.Preview = e.preview
		a.Diff = true
	}
	return a
}

func TestChangePreviewUsesExecutedCandidate(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "verify", "purpose": "check"}), "Done"}}
	e := &previewExecutor{preview: "--- a\n+++ a\n@@ -1,1 +1,1 @@\n-old\\r\n+new\\r\n"}
	j := &journal{}
	r := fixture(g, &e.executor, j)
	r.Tools = e
	r.Settings.Permission = policy.AllowAll
	_, err := r.Run(context.Background(), "replace text", false, approve, emit)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, event := range j.events {
		if event.Kind == "tool_succeeded" && event.Call.Name == "write_file" {
			found = true
			if event.ChangePreview != e.preview {
				t.Fatalf("preview=%q", event.ChangePreview)
			}
			if strings.Contains(string(event.Output), "old") {
				t.Fatal("display duplicated into model result")
			}
			if err := event.Check(); err != nil {
				t.Fatal(err)
			}
		}
	}
	if !found {
		t.Fatal("missing applied preview")
	}
}

func TestChangePreviewLimitPreservesUTF8(t *testing.T) {
	raw := strings.Repeat("한글", maxChangePreviewBytes)
	shown := boundedChangePreview(raw)
	if len(shown) > maxChangePreviewBytes || !utf8.ValidString(shown) || !strings.Contains(shown, "일부 생략") {
		t.Fatal("invalid bounded preview")
	}
}

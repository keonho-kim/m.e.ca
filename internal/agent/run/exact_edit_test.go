// Purpose: Verify shell inspection followed by exact-text editing preserves approval and line positions.
// Pattern: Agent and filesystem integration test.
// Usage: go test ./internal/agent/run -run TestExactEditDialogue.
// Related: internal/tools/local/textedit/ranges.go, internal/agent/run/action.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExactEditDialogue(t *testing.T) {
	r, _, project, _, _ := realBatchRunner(t, nil)
	if err := os.WriteFile(filepath.Join(project, "a"), []byte("first\nsecond\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	script := &generator{replies: []string{
		nativeCall("execute_command", map[string]any{"command": "cat a"}),
		nativeCall("edit_file", map[string]any{"path": "a", "edits": []map[string]any{{"old_text": "second\n", "new_text": "second changed\n"}}}),
		nativeCall("execute_command", map[string]any{"command": "test -f a"}), "Updated a at line 2 and checked it",
	}}
	r.Generator = script
	r.Settings.Permission = policy.Approve
	result, err := r.Run(context.Background(), "change the second line", approve, emit)
	if err != nil || result.Status != Answered || len(script.requests) != 4 {
		t.Fatalf("%+v %v", result, err)
	}
	content, err := os.ReadFile(filepath.Join(project, "a"))
	if err != nil || string(content) != "first\nsecond changed\nlast\n" {
		t.Fatalf("%q %v", content, err)
	}
	var text strings.Builder
	for _, message := range script.requests[len(script.requests)-1].Messages {
		text.WriteString(message.Content)
	}
	if !strings.Contains(text.String(), "NewStartLine") || !strings.Contains(text.String(), "NewEndLine") {
		t.Fatal("final line positions missing")
	}
}

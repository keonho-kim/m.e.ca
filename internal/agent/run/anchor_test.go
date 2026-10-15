// Purpose: Verify the model reads short line references and completes an anchored edit through native calls and approval.
// Pattern: Agent and filesystem integration test.
// Usage: go test ./internal/agent/run -run TestAnchoredEditDialogue.
// Related: internal/agent/tool/arguments.go, internal/tools/local/anchors/edit.go, internal/agent/run/action.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/policy"
	"github.com/keonho-kim/v-code/internal/tools/local"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type anchorGenerator struct {
	script       *generator
	journal      *journal
	anchorFields int
}

func (g *anchorGenerator) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	return g.script.Generate(ctx, r, func(event modelapi.Event) error {
		for i := range event.Calls {
			call := &event.Calls[i]
			if call.Name != "edit_file" {
				continue
			}
			var args map[string]any
			if err := json.Unmarshal(call.Arguments, &args); err != nil {
				return err
			}
			for _, message := range g.journal.messages {
				if message.Role != "tool" || message.ToolName != "read_file" {
					continue
				}
				var read struct{ Content string }
				if err := json.Unmarshal([]byte(message.Content), &read); err != nil {
					return err
				}
				for _, line := range strings.Split(read.Content, "\n") {
					if strings.HasPrefix(line, "2#") {
						anchor, _, _ := strings.Cut(line, " | ")
						args["start_anchor"], args["end_anchor"] = anchor, anchor
						g.anchorFields += 2
					}
				}
			}
			if args["start_anchor"] == nil {
				return fmt.Errorf("read anchors missing")
			}
			body, err := json.Marshal(args)
			if err != nil {
				return err
			}
			call.Arguments = body
		}
		return emit(event)
	})
}
func TestAnchoredEditDialogue(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "a"), []byte("first\nsecond\nlast\n"), 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := local.NewExecutor(project, home, filepath.Join(home, ".v"), func(s string) string { return s })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	script := &generator{purposes: []string{"work", "check"}, replies: []string{nativeCall("read_file", map[string]any{"path": "a", "offset": 0, "limit": 0}), nativeCall("edit_file", map[string]any{"path": "a", "new_text": "second changed\n"}), nativeCall("execute_command", map[string]any{"command": "test -f a"}), "Updated a at line 2 and checked it"}}
	journal := &journal{}
	model := &anchorGenerator{script: script, journal: journal}
	runner := fixture(script, &executor{}, journal)
	runner.Generator = model
	runner.Tools = tools
	runner.Settings.Scope = policy.System
	result, err := runner.Run(context.Background(), "change the second line", false, approve, emit)
	if err != nil || result.Status != Verified || model.anchorFields != 2 || len(script.reviewRequests) != 1 {
		t.Fatalf("result=%+v fields=%d err=%v", result, model.anchorFields, err)
	}
	content, err := os.ReadFile(filepath.Join(project, "a"))
	if err != nil || string(content) != "first\nsecond changed\nlast\n" {
		t.Fatalf("%q %v", content, err)
	}
	var contextText strings.Builder
	for _, message := range script.requests[len(script.requests)-1].Messages {
		contextText.WriteString(message.Content)
	}
	if !strings.Contains(contextText.String(), "NewStartLine") || !strings.Contains(contextText.String(), "NewEndLine") {
		t.Fatal("model cannot report the resulting lines")
	}
}

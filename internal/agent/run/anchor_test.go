// Purpose: Verify the model reads short line references and completes an anchored edit through PFC and approval.
// Pattern: Agent and filesystem integration test.
// Usage: go test ./internal/agent/run -run TestAnchoredEditDialogue.
// Related: internal/agent/pfc/pfc.go, internal/tools/local/anchors/edit.go, internal/agent/run/action.go.
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
	question := r.Messages[len(r.Messages)-1].Content
	if strings.Contains(question, "ARGUMENT edit_file.start_anchor:") || strings.Contains(question, "ARGUMENT edit_file.end_anchor:") {
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
					g.anchorFields++
					return emit(modelapi.Event{Delta: anchor + "\n", Done: true})
				}
			}
		}
		return fmt.Errorf("read anchors missing from model observations")
	}
	return g.script.Generate(ctx, r, emit)
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
	script := &generator{purposes: []string{"work", "check"}, replies: []string{`{"tool":"read_file"}`, "a", "0", "0", `{"tool":"edit_file"}`, "a", "second changed\n", `{"tool":"execute_command"}`, "test -f a", "DONE\nUpdated a at line 2 and checked it"}}
	journal := &journal{}
	model := &anchorGenerator{script: script, journal: journal}
	runner := fixture(script, &executor{}, journal)
	runner.Generator = model
	runner.Tools = tools
	runner.Settings.Scope = policy.System
	result, err := runner.Run(context.Background(), "change the second line", false, approve, emit)
	if err != nil || result.Status != Verified || model.anchorFields != 2 || len(script.reviewRequests) != 3 {
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

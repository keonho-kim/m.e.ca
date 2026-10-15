// Purpose: Verify ordinary commands retain unchanged file facts and invalidate actual changes.
// Pattern: Filesystem integration regressions with scripted model responses.
// Usage: go test ./internal/agent/run -run TestFileFacts.
// Related: internal/agent/run/file_facts.go, internal/tools/local/versions.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"strings"
	"testing"
)

func TestFileFactsCompareActualVersionsAfterCommands(t *testing.T) {
	for _, tc := range []struct {
		command string
		stale   bool
	}{{"ls", false}, {"printf changed > a", true}} {
		t.Run(tc.command, func(t *testing.T) {
			r, _, _, _, _ := realBatchRunner(t, nil)
			g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "initial"}), nativeCall("execute_command", map[string]any{"command": tc.command}), "Observed the files and command output."}}
			g.completion = func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
				text := request.Messages[len(request.Messages)-1].Content
				var input completion.Input
				if err := json.Unmarshal([]byte(text[strings.IndexByte(text, '\n')+1:]), &input); err != nil {
					return err
				}
				if len(input.State.Receipts) != 2 || input.State.Receipts[0].Stale != tc.stale {
					t.Fatalf("file freshness=%+v", input.State.Receipts)
				}
				return emit(modelapi.Event{Delta: `{"decision":"complete","reason":"The requested operation and observation finished","evidence":["call_2"]}`, Done: true})
			}
			r.Generator = g
			result, err := r.Run(context.Background(), "Create a, then perform the specified command", approve, emit)
			if err != nil || result.Status != Answered {
				t.Fatalf("%+v %v", result, err)
			}
		})
	}
}

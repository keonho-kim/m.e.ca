// Purpose: Verify a test-file review can inspect an unexecuted sibling implementation in a real batch.
// Pattern: Filesystem and scripted-provider integration regression.
// Usage: go test ./internal/agent/run -run TestBatchReviewPendingImplementation.
// Related: internal/agent/run/batch_wave.go, internal/agent/conclave/proposed_context.go.
package run

import (
	"context"
	"encoding/json"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type pendingReviewModel struct {
	base          *parallelModel
	root          string
	t             *testing.T
	contextRounds int
}

func (g *pendingReviewModel) Generate(ctx context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	text := r.Messages[len(r.Messages)-1].Content
	if !strings.HasPrefix(text, "REVIEW CANDIDATE ") {
		return g.base.Generate(ctx, r, emit)
	}
	var packet struct {
		Files        []toolapi.FileDelta
		Context      []json.RawMessage
		PendingPaths []string
	}
	start := strings.Index(text, "\n")
	decoder := json.NewDecoder(strings.NewReader(text[start+1:]))
	if err := decoder.Decode(&packet); err != nil {
		g.t.Fatal(err)
	}
	if strings.HasSuffix(packet.Files[0].Path, "test_calculator.py") {
		if _, err := os.Stat(filepath.Join(g.root, "calculator.py")); !os.IsNotExist(err) {
			g.t.Fatal("sibling executed before all reviews")
		}
		if len(packet.Context) == 0 {
			g.contextRounds++
			body, _ := json.Marshal(map[string]any{"decision": "needs_context", "context_requests": []any{map[string]any{"path": packet.PendingPaths[0], "offset": 0, "limit": 8192, "reason": strings.Repeat("Inspect the frozen implementation. ", 12)}}})
			return emit(modelapi.Event{Delta: string(body), Done: true})
		}
		if !strings.Contains(string(packet.Context[0]), `"executed":false`) || !strings.Contains(string(packet.Context[0]), "sum_positive") {
			g.t.Fatal("pending implementation context missing")
		}
	}
	return emit(modelapi.Event{Delta: `{"decision":"pass"}`, Done: true})
}
func TestBatchReviewPendingImplementation(t *testing.T) {
	calls := []toolapi.Call{
		batchCall("impl", "write_file", map[string]any{"path": "calculator.py", "content": "def sum_positive(xs):\n    return sum(x for x in xs if x > 0)\n"}),
		batchCall("test", "write_file", map[string]any{"path": "test_calculator.py", "content": "from calculator import sum_positive\nassert sum_positive([1,-1]) == 1\n"}),
	}
	r, _, root, _, base := realBatchRunner(t, calls)
	g := &pendingReviewModel{base: base, root: root, t: t}
	r.Generator = g
	result, err := r.Run(context.Background(), "create implementation and tests", approve, emit)
	if err != nil || result.Status != Answered || g.contextRounds != 1 {
		t.Fatalf("%+v context=%d %v", result, g.contextRounds, err)
	}
	for _, file := range []string{"calculator.py", "test_calculator.py"} {
		if _, err := os.Stat(filepath.Join(root, file)); err != nil {
			t.Fatal(err)
		}
	}
}

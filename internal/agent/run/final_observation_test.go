// Purpose: Keep final worker facts current when a file changes during answer generation.
// Pattern: Filesystem and generation-boundary regression test.
// Usage: go test ./internal/agent/run -run TestFinalFacts.
// Related: internal/agent/run/finish_response.go, internal/agent/run/file_facts.go.
package run

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

type changedAnswer struct {
	generator
	path string
}

func (g *changedAnswer) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	if err := g.generator.Generate(ctx, request, emit); err != nil {
		return err
	}
	if len(g.requests) == 2 {
		return os.WriteFile(g.path, []byte("external change"), 0600)
	}
	return nil
}

func TestFinalFactsRefreshAfterAnswerGeneration(t *testing.T) {
	r, _, root, _, _ := realBatchRunner(t, nil)
	g := &changedAnswer{generator: generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "initial"}), "Created a; checks were not run."}}, path: filepath.Join(root, "a")}
	r.Generator = g
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	want := fmt.Sprintf("%x", sha256.Sum256([]byte("external change")))
	if err != nil || len(result.Facts.Files) != 1 || result.Facts.Files[0].Version != want || len(g.requests) != 2 {
		t.Fatalf("final observations were stale: %+v calls=%d err=%v", result, len(g.requests), err)
	}
}

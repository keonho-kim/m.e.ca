// Purpose: Stop uncertain skipped-result commits without submitting the same record again.
// Pattern: Scripted failure at the journal boundary with observed commit attempts.
// Usage: go test ./internal/agent/run -run TestCodeDoesNotRetryUncertainSkippedResult.
// Related: internal/agent/run/code_dispatch.go, internal/agent/run/batch_feedback.go.
package run

import (
	"context"
	"errors"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

type uncertainSkipJournal struct {
	*journal
	attempts int
}

func (j *uncertainSkipJournal) Commit(ctx context.Context, session string, entries []runrecord.Entry) error {
	for _, entry := range entries {
		if event := entry.Event; event != nil && event.Kind == "tool_failed" && event.Call != nil && event.Call.Name == "write_file" {
			j.attempts++
			return errors.New("uncertain skipped-result commit")
		}
	}
	return j.journal.Commit(ctx, session, entries)
}

func TestCodeDoesNotRetryUncertainSkippedResult(t *testing.T) {
	g := &generator{replies: []string{codeCall(`return await Promise.all([
		tools.inspect_fixture({path:"a"}), tools.write_file({path:"b",content:"must not run"})
	]);`)}}
	e := &executor{outcomes: []toolapi.Result{{Failed: true, Content: []byte(`{}`)}}}
	j := &uncertainSkipJournal{journal: &journal{}}
	r := fixture(g, e, j.journal)
	r.Journal = j
	_, err := r.Run(t.Context(), "stop on failed read", approve, emit)
	if !errors.Is(err, errRuntimeRecord) || j.attempts != 1 || e.executed != 1 {
		t.Fatalf("commit attempts=%d effects=%d error=%v", j.attempts, e.executed, err)
	}
}

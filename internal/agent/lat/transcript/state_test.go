// Purpose: Verify bounded local context, atomic publication and explicit phase cleanup.
// Pattern: Journal boundary tests.
// Usage: go test ./internal/agent/lat/transcript.
// Related: internal/agent/lat/transcript/journal.go.
package transcript

import (
	"context"
	"errors"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type failingCommitter struct{ err error }

func (c *failingCommitter) Commit(context.Context, string, []runrecord.Entry) error { return c.err }

func TestLocalContextPublishesOnlyAfterParentCommit(t *testing.T) {
	parent := &failingCommitter{err: errors.New("disk full")}
	journal, err := New(parent, "session", "parent_run", "critique")
	if err != nil {
		t.Fatal(err)
	}
	defer journal.Close()
	message := modelapi.Message{Role: "user", Content: "request"}
	if err = journal.Commit(context.Background(), "session", []runrecord.Entry{{Message: &message}}); !errors.Is(err, parent.err) {
		t.Fatalf("parent error lost: %v", err)
	}
	if len(journal.messages) != 0 {
		t.Fatal("failed parent commit published local messages")
	}
	parent.err = nil
	if err = journal.Commit(context.Background(), "session", []runrecord.Entry{{Message: &message}}); err != nil {
		t.Fatal(err)
	}
	if messages, err := journal.ContextMessages(context.Background(), "session", 0, 1, true); err == nil || messages != nil {
		t.Fatalf("context overflow silently truncated: %+v %v", messages, err)
	}
	if messages, err := journal.ContextMessages(context.Background(), "session", 0, 1024, true); err != nil || len(messages) != 1 {
		t.Fatalf("bounded read changed stored history: %+v %v", messages, err)
	}
	if err = journal.Commit(context.Background(), "other", []runrecord.Entry{{Message: &message}}); err == nil {
		t.Fatal("cross-session write accepted")
	}
	retained := journal.messages
	journal.Close()
	journal.Close()
	if journal.parent != nil || journal.messages != nil || journal.session != "" || journal.scope != (runrecord.LATRun{}) || retained[0].message.Content != "" {
		t.Fatal("Close retained parent or native history references")
	}
}

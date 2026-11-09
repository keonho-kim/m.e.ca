// Purpose: Keep external historical context available without duplicating it in the execution journal.
// Pattern: Request dataflow contract test.
// Usage: go test ./internal/agent/run -run TestClientContext.
// Related: internal/agent/context/request.go, internal/commands/server/submission_session.go.
package run

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
	"testing"
)

func TestClientContextIsVisibleButNotPersistedAsExecution(t *testing.T) {
	g, j := &generator{replies: []string{"done"}}, &journal{}
	r := fixture(g, &executor{}, j)
	r.ClientContext = []modelapi.Message{{Role: "user", Content: "external-history-canary"}, {Role: "assistant", Content: "client-provided-reply"}}
	if _, err := r.Run(t.Context(), "new question", approve, emit); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range g.requests[0].Messages {
		found = found || strings.Contains(m.Content, "external-history-canary")
	}
	if !found {
		t.Fatal("external history absent from model context")
	}
	for _, m := range j.messages {
		if strings.Contains(m.Content, "external-history-canary") || strings.Contains(m.Content, "client-provided-reply") {
			t.Fatal("external context was persisted as engine execution")
		}
	}
}

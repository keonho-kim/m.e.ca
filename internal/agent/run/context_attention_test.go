// Purpose: Preserve all public observations while requesting extra inference only for actionable attention.
// Pattern: Scripted generation boundary tests.
// Usage: go test ./internal/agent/run -run TestPeerAttention.
// Related: internal/agent/run/context_inbox.go, internal/agent/run/loop.go.
package run

import (
	"context"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/source"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestPeerAttentionControlsContinuationWithoutChangingUserObjective(t *testing.T) {
	for _, reconsider := range []bool{false, true} {
		t.Run(map[bool]string{false: "ordinary-progress", true: "public-feedback"}[reconsider], func(t *testing.T) {
			g := &generator{replies: []string{"Implemented the requested change", "Considered the public feedback"}}
			j := &contextJournalFixture{}
			r := fixture(g, &executor{}, &j.journal)
			r.Journal = j
			r.PollContext = func(context.Context) (ContextBatch, error) {
				if len(g.requests) == 1 && j.cursor == 0 {
					return ContextBatch{Cursor: 7, Reconsider: reconsider, Messages: []modelapi.Message{{ID: "peer:Ren/7", Source: &source.Public{Author: "Ren", Kind: "team_message", Sequence: 7}, Role: "system", Content: "Public conversation — Ren\nMy tests cover the existing 423 response."}}}, nil
				}
				return ContextBatch{}, nil
			}
			result, err := r.Run(context.Background(), "Implement the authentication endpoint", approve, emit)
			want := 1
			if reconsider {
				want = 2
			}
			if err != nil || result.Status != runrecord.Answered || len(g.requests) != want || j.cursor != 7 {
				t.Fatal("public admission and inference were coupled", result, len(g.requests), err)
			}
			found := false
			for _, message := range j.messages {
				found = found || strings.Contains(message.Content, "existing 423 response")
			}
			if !found {
				t.Fatal("an observation was omitted because it did not wake the agent")
			}
			if reconsider {
				found = false
				for _, message := range g.requests[1].Messages {
					found = found || message.Role == "user" && message.Content == "Implement the authentication endpoint"
				}
				if !found {
					t.Fatal("peer opinion replaced the user's request")
				}
			}
		})
	}
}

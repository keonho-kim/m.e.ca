// Purpose: Reuse owned conversation reads for token totals without reopening SQLite.
// Pattern: Resource-reuse regression test with consumer-owned history.
// Usage: go test ./internal/commands/server -run TestSessionUsage.
// Related: internal/commands/server/session_usage.go, internal/commands/server/fixture_test.go.
package server

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/model"
	"testing"
)

type usageHistory struct{ detailHistory }

func (*usageHistory) TokenUsage(context.Context, string) (model.TokenUsage, error) {
	return model.TokenUsage{Input: 17, Output: 9}, nil
}

func TestSessionUsageReusesOwnedHistory(t *testing.T) {
	s := detailFixture(&usageHistory{}, nil)
	defer s.Close()
	s.options.Usage = func(context.Context, string) (model.TokenUsage, error) {
		t.Error("reopened an already owned conversation")
		return model.TokenUsage{}, nil
	}
	value, err := s.SessionUsage(t.Context(), s.conversation.Runtime.Session.ID)
	if err != nil || value.Input != 17 || value.Output != 9 {
		t.Fatal(value, err)
	}
}

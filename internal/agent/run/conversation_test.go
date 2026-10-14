// Purpose: Verify conversation routes directly to answers and only selected tools enter Goal analysis.
// Pattern: Runtime routing contract tests.
// Usage: go test ./internal/agent/run -run TestConversationRouting.
// Related: internal/agent/run/conversation.go, internal/agent/run/goal.go.
package run

import (
	"context"
	"strings"
	"testing"

	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestConversationRoutingAnswersWithoutGoalAnalysis(t *testing.T) {
	for _, input := range []string{"Hello", "안녕하세요", "Explain what a compiler does"} {
		g := &generator{replies: []string{"Hello! How can I help?"}}
		e, j := &executor{}, &journal{}
		result, err := fixture(g, e, j).RunGoal(context.Background(), input, approve, emit)
		if err != nil || result.Status != Answered || len(g.requests) != 1 || e.prepared != 0 {
			t.Fatalf("input=%q result=%+v calls=%d err=%v", input, result, len(g.requests), err)
		}
		for _, event := range j.events {
			if event.Goal != nil || strings.HasPrefix(event.Kind, "goal_analysis") {
				t.Fatal("ordinary conversation entered Goal analysis")
			}
		}
	}
}

func TestConversationRoutingWorkSurvivesMalformedDecomposition(t *testing.T) {
	g := &generator{replies: []string{`{"tool":"write_file"}`, "bad", "bad", "bad", `{"tool":"write_file"}`, "README.md", "# Project", `{"tool":"execute_command"}`, "test -s README.md", "DONE\nCreated README.md"}, purposes: []string{"check"}}
	e, j := &executor{approval: true}, &journal{}
	approvals := 0
	result, err := fixture(g, e, j).RunGoal(context.Background(), "Create README.md here", func(ctx context.Context, request toolapi.Approval) (bool, error) { approvals++; return true, nil }, emit)
	if err != nil || result.Status != Verified || e.executed != 2 || approvals != 2 {
		t.Fatalf("result=%+v executed=%d err=%v", result, e.executed, err)
	}
}

func TestConversationRoutingStreamsPlainAnswer(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	partial := ""
	g := &chunkGenerator{chunks: []string{"Hel", "lo there"}}
	g.after = func(index int) {
		if index == 0 && partial != "Hel" {
			t.Fatalf("plain answer did not stream: %q", partial)
		}
	}
	r.Generator = g
	result, err := r.RunGoal(context.Background(), "Hello", approve, func(event Event) error {
		if event.Kind == "stream" {
			partial = event.Text
		}
		return nil
	})
	if err != nil || result.Status != Answered || partial != "Hello there" || result.Text != partial {
		t.Fatalf("result=%+v partial=%q err=%v", result, partial, err)
	}
}

// Purpose: Verify file reversals and repeated observations trigger bounded recovery without replaying mutations.
// Pattern: Runtime progress state tests.
// Usage: go test ./internal/agent/run -run TestProgress.
// Related: internal/agent/run/progress.go, internal/agent/run/action.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/policy"
	"strings"
	"testing"
)

func TestProgressFreshVerificationIsNotARepeat(t *testing.T) {
	g := &generator{purposes: []string{"check", "check"}, replies: []string{`{"tool":"execute_command"}`, "verify", `{"tool":"write_file"}`, "a", "new", `{"tool":"execute_command"}`, "verify", "DONE\nVerified"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	r.Settings.Scope = policy.System
	r.Limits = DefaultLimits()
	r.Limits.StallIterations = 1
	result, err := r.Run(context.Background(), "change and verify", false, approve, emit)
	if err != nil || result.Status != Verified {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, event := range j.events {
		if event.Kind == "recovery" {
			t.Fatal("fresh verification was treated as no progress")
		}
	}
}

func TestProgressRecoveryOffersSpecificNextAction(t *testing.T) {
	g := &generator{replies: []string{"CONTINUE\ntry", "CONTINUE\ntry", "CONTINUE\ntry", "WAIT\nNeed another approach"}}
	j := &journal{}
	r := fixture(g, &executor{}, j)
	result, err := r.Run(context.Background(), "inspect", false, approve, emit)
	if err != nil || result.Status != NeedsInput {
		t.Fatalf("recovery=%+v %v", result, err)
	}
	last := g.requests[len(g.requests)-1]
	if !strings.Contains(last.Messages[len(last.Messages)-1].Content, "Recovery 1/2") {
		t.Fatal("recovery choices missing from context")
	}
}

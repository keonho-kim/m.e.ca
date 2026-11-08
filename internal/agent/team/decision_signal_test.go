// Purpose: Share instruction-generation invalidation without per-tool channels or goroutines.
// Pattern: Deterministic signal and decision freshness tests.
// Usage: Team tests bind waiters to actual model-reviewed user input.
// Related: internal/agent/team/decision.go, internal/agent/team/publication.go.
package team

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	"errors"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

func TestInstructionSignalIsSharedUntilTheNextUserMessage(t *testing.T) {
	r := testRoom(t)
	p := &Peer{ID: "Hana", room: r}
	first, err := p.UserChanges()
	if err != nil {
		t.Fatal(err)
	}
	same, err := p.UserChanges()
	if err != nil || same != first {
		t.Fatal("repeated admission allocated another generation", err)
	}
	if _, err := r.publish(channel.Message{From: "user", Text: "Keep existing files"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-first:
	default:
		t.Fatal("new instruction did not invalidate waiting work")
	}
	var failure *faults.Failure
	if _, err := p.UserChanges(); !errors.As(err, &failure) || failure.Code != diagcatalog.InstructionChanged {
		t.Fatal("receiving a new generation bypassed model review", err)
	}
	p.Inbox(r.ctx)
	p.BeginDecision()
	next, err := p.UserChanges()
	if err != nil || next == first {
		t.Fatal("fresh model decision did not receive a new signal", err)
	}
	select {
	case <-next:
		t.Fatal("fresh decision was already invalidated")
	default:
	}
}

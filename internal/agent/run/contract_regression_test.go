// Purpose: Verify failed calls, exact final drafts and response streaming obey completion contracts.
// Pattern: Scripted runtime regression tests.
// Usage: go test ./internal/agent/run -run TestContract.
// Related: internal/agent/run/completion_review.go, internal/agent/run/generate.go.
package run

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/keonho-kim/v-code/internal/agent/completion"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestContractInvalidBatchCannotClaimCompletion(t *testing.T) {
	calls := []toolapi.Call{batchCall("bad", "write_file", map[string]any{"path": "a"}), batchCall("read", "read_file", map[string]any{"path": "a"})}
	r, _, _, j, _ := realBatchRunner(t, calls)
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	if err != nil || result.Status != Stopped || result.Text == "Completed requested operations" {
		t.Fatalf("failed batch accepted as completion: %+v %v", result, err)
	}
	for _, e := range j.events {
		if e.Kind == "tool_started" {
			t.Fatal("invalid batch executed")
		}
	}
}
func TestContractEntireDraftIsReviewed(t *testing.T) {
	tail := "The requested file was read. No deployment was performed."
	draft := strings.Repeat("Observed file contents. ", 500) + tail
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), draft}}
	g.completion = func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		text := r.Messages[len(r.Messages)-1].Content
		var packet completion.Input
		if err := json.Unmarshal([]byte(text[strings.IndexByte(text, '\n')+1:]), &packet); err != nil {
			return err
		}
		if packet.Draft != draft {
			t.Fatalf("reviewed %d of %d draft bytes", len(packet.Draft), len(draft))
		}
		return emit(modelapi.Event{Delta: `{"decision":"complete","reason":"The draft matches the read","evidence":["call_1"]}`, Done: true})
	}
	result, err := fixture(g, &executor{}, &journal{}).Run(context.Background(), "Read a", approve, emit)
	if err != nil || result.Text != draft {
		t.Fatalf("final differs: %v", err)
	}
}
func TestContractPostToolResponseStreams(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = &chunkGenerator{chunks: []string{"Hello", " world"}}
	visible := 0
	s := execution{runner: r, limits: DefaultLimits(), usage: &Usage{}, system: "rules", checkpoint: Checkpoint{Phase: "decision", Work: completion.State{Receipts: []completion.Receipt{{ID: "r", Tool: "read_file", Success: true}}}}, emit: func(e Event) error {
		if e.Kind == "stream" && e.Text != "" {
			visible++
		}
		return nil
	}}
	reply, err := s.askResponse(context.Background(), "Answer now", inference.Agent, modelapi.MaxResponseBytes)
	if err != nil || reply.Text != "Hello world" || visible != 2 {
		t.Fatalf("stream events=%d error=%v", visible, err)
	}
}

func TestContractCancelledReviewRetainsUnfinishedAnswer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	draft := "Read a; the requested follow-up still needs confirmation."
	g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a"}), draft}}
	g.completion = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
		cancel()
		return context.Canceled
	}
	j := &journal{}
	_, err := fixture(g, &executor{}, j).Run(ctx, "Read a", approve, emit)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel=%v", err)
	}
	found := false
	for _, m := range j.messages {
		if m.Role == "system" && strings.Contains(m.Content, draft) {
			found = true
		}
	}
	if !found {
		t.Fatal("unfinished streamed answer was lost during review cancellation")
	}
}

type malformedWorkGenerator struct{ failed bool }

func (g *malformedWorkGenerator) Generate(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
	if handled, err := completionFixture(r, emit); handled {
		return err
	}
	if !g.failed {
		g.failed = true
		return &modelapi.ToolCallError{Detail: "invalid generated arguments"}
	}
	return emit(modelapi.Event{Delta: "Completed the requested file.", Done: true})
}
func TestContractMalformedToolResponseCannotClaimCompletion(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator = &malformedWorkGenerator{}
	result, err := r.Run(context.Background(), "Create a", approve, emit)
	if err != nil || result.Status != Stopped || result.Text == "Completed the requested file." {
		t.Fatalf("%+v %v", result, err)
	}
}

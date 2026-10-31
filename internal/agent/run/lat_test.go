// Purpose: Verify optional independent review returns advisory feedback without changing the normal loop.
// Pattern: Scripted runtime behavior tests.
// Usage: go test ./internal/agent/run -run TestLAT.
// Related: internal/agent/run/lat.go, internal/agent/run/run_test.go.
package run

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type latModel struct {
	main     *generator
	review   func(context.Context, modelapi.Request, func(modelapi.Event) error) error
	reviews  []modelapi.Request
	released bool
}

// A real client has separate main-run encoder state; review uses the raw generator.
func (m *latModel) NewRunGenerator() (modelapi.Generator, func()) {
	return m.main, func() { m.released = true }
}
func (m *latModel) Generate(ctx context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	m.reviews = append(m.reviews, request)
	if m.review != nil {
		return m.review(ctx, request, emit)
	}
	return emit(modelapi.Event{Delta: "NO_FINDINGS", Done: true})
}

func latFixture(replies ...string) (Runner, *latModel, *executor, *journal) {
	main, tools, store := &generator{replies: replies}, &executor{}, &journal{}
	r := fixture(main, tools, store)
	model := &latModel{main: main}
	r.Generator, r.LAT = model, true
	r.ReviewMemory = &journalMemory{journal: store}
	return r, model, tools, store
}

func TestLATOffAndOrdinaryChatPreserveCalls(t *testing.T) {
	for _, work := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			replies := []string{"Hello"}
			if work {
				replies = []string{nativeCall("write_file", map[string]any{"path": "a", "content": "one"}), "Changed a"}
			}
			r, model, tools, _ := latFixture(replies...)
			r.LAT = enabled
			result, err := r.Run(context.Background(), "request", approve, emit)
			wantReviews := 0
			if work && enabled {
				wantReviews = 1
			}
			if err != nil || result.Status != runrecord.Answered || len(model.reviews) != wantReviews || len(model.main.requests) != len(replies) || !model.released {
				t.Fatalf("enabled=%t work=%t result=%+v main=%d review=%d err=%v", enabled, work, result, len(model.main.requests), len(model.reviews), err)
			}
			if !work && tools.executed != 0 {
				t.Fatal("chat executed tools")
			}
		}
	}
}

func TestLATFeedbackIsIsolatedAndNotRecursivelyReviewed(t *testing.T) {
	r, model, tools, store := latFixture(
		nativeCall("write_file", map[string]any{"path": "api.py", "content": "original"}), "Initial draft",
		nativeCall("write_file", map[string]any{"path": "api.py", "content": "corrected"}), "Finished correction",
	)
	store.messages = []modelapi.Message{{Role: "user", Content: "Use a simple API"}, {Role: "assistant", Content: "PRIVATE_PARENT_ANSWER", Reasoning: "PRIVATE_PARENT_THINKING"}}
	model.review = func(_ context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
		if len(request.Tools) != 0 || len(request.Messages) != 2 || request.MaxOutput != 4096 || request.Temperature == nil || *request.Temperature != 0.2 {
			t.Fatal("review inherited main capabilities or inference parameters")
		}
		all := request.Messages[0].Content + request.Messages[1].Content
		for _, text := range []string{"PRIVATE_PARENT_ANSWER", "PRIVATE_PARENT_THINKING"} {
			if strings.Contains(all, text) {
				t.Fatal("parent discussion leaked to reviewer")
			}
		}
		for _, text := range []string{"Use a simple API", "Implement api.py", "api.py", "Initial draft", r.ProjectInstructions} {
			if !strings.Contains(all, text) {
				t.Fatalf("review lost task evidence: %s", text)
			}
		}
		return send(modelapi.Event{Delta: "- api.py: requested behavior is missing. Add the handler.", Done: true})
	}
	result, err := r.Run(context.Background(), "Implement api.py", approve, emit)
	if err != nil || result.Text != "Finished correction" || tools.executed != 2 || len(model.reviews) != 1 || recordedModelCalls(store) != 5 {
		t.Fatalf("review repair loop: %+v executed=%d reviews=%d err=%v", result, tools.executed, len(model.reviews), err)
	}
	context := model.main.requests[2].Messages
	found := false
	for _, message := range context {
		found = found || strings.Contains(message.Content, "requested behavior is missing")
	}
	if !found {
		t.Fatal("main agent did not receive review feedback")
	}
}

func TestLATUnavailableIsExplicitAndAdvisory(t *testing.T) {
	for _, journalFailure := range []bool{false, true} {
		r, model, _, store := latFixture(nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Observed a")
		model.review = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			return errors.New("review service unavailable after retries")
		}
		if journalFailure {
			store.failKind = "lat_unavailable"
		}
		result, err := r.Run(context.Background(), "Inspect a", approve, emit)
		if journalFailure {
			if !errors.Is(err, errRuntimeRecord) {
				t.Fatal("lost durable failure", err)
			}
			continue
		}
		if err != nil || result.Status != runrecord.Answered || len(model.main.requests) != 2 {
			t.Fatalf("review became a completion gate: %+v %v", result, err)
		}
		found := false
		for _, event := range store.events {
			if event.Kind == "lat_clean" {
				t.Fatal("failed review reported clean")
			}
			found = found || event.Kind == "lat_unavailable" && strings.Contains(event.Text, "unavailable after retries")
		}
		if !found {
			t.Fatal("review failure was hidden")
		}
	}
}

func TestLATCancellationStopsBeforeFurtherWork(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	r, model, tools, store := latFixture(nativeCall("write_file", map[string]any{"path": "a", "content": "one"}), "Draft")
	model.review = func(ctx context.Context, _ modelapi.Request, _ func(modelapi.Event) error) error {
		cancel()
		return ctx.Err()
	}
	result, err := r.Run(ctx, "Create a", approve, emit)
	if !errors.Is(err, context.Canceled) || result.Status != runrecord.Interrupted || tools.executed != 1 || !model.released {
		t.Fatalf("cancelled review: %+v %v", result, err)
	}
	last := store.messages[len(store.messages)-1]
	if last.Role != "assistant" || last.Content != "Draft" || !last.Incomplete || len(last.ReasoningState) != 0 {
		t.Fatalf("interrupted review lost the visible main draft: %+v", last)
	}
}

func TestLATSteeringDiscardsOldFeedback(t *testing.T) {
	r, model, _, store := latFixture(nativeCall("inspect_fixture", map[string]any{"path": "a"}), "Old draft", nativeCall("inspect_fixture", map[string]any{"path": "b"}), "Follow the new request")
	var ready atomic.Bool
	r.PollSteering = func() []string {
		if !ready.Swap(false) {
			return nil
		}
		return []string{"New user request"}
	}
	model.review = func(_ context.Context, _ modelapi.Request, send func(modelapi.Event) error) error {
		if len(model.reviews) == 1 {
			ready.Store(true)
			return send(modelapi.Event{Delta: "STALE_REVIEW_FINDING", Done: true})
		}
		return send(modelapi.Event{Delta: "NO_FINDINGS", Done: true})
	}
	result, err := r.Run(context.Background(), "Old request", approve, emit)
	if err != nil || result.Text != "Follow the new request" || len(model.reviews) != 2 {
		t.Fatalf("steering: %+v %v", result, err)
	}
	for _, message := range model.main.requests[2].Messages {
		if strings.Contains(message.Content, "STALE_REVIEW_FINDING") {
			t.Fatal("old review affected a new request")
		}
	}
	discarded := false
	for _, event := range store.events {
		discarded = discarded || event.Kind == "lat_discarded"
	}
	if !discarded {
		t.Fatal("review discard was not recorded")
	}
}

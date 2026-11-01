// Purpose: Verify the native LAT phases, isolated context and one advisory repair.
// Pattern: Scripted runtime behavior tests.
// Usage: go test ./internal/agent/run -run TestLAT.
// Related: internal/agent/run/lat.go, internal/agent/run/lat_fixture_test.go.
package run

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestLATOffPreservesOrdinaryCalls(t *testing.T) {
	for _, replies := range [][]string{
		{"Hello"},
		{nativeCall("write_file", map[string]any{"path": "a", "content": "one"}), "Changed a"},
	} {
		r, model, tools, _ := latFixture(replies...)
		r.LAT = false
		result, err := r.Run(context.Background(), "request", approve, emit)
		if err != nil || result.Status != runrecord.Answered || len(model.requests["main"]) != len(replies) || model.created != 1 || model.released != 1 || len(tools.views) != 0 {
			t.Fatalf("LAT off: result=%+v scopes=%d/%d calls=%v err=%v", result, model.created, model.released, model.scopes, err)
		}
	}
}

func TestLATNativePhasesInspectRepairAndRemainIsolated(t *testing.T) {
	r, model, tools, store := latFixture(
		nativeCall("write_file", map[string]any{"path": "api.py", "content": "original"}), "Initial draft",
		nativeCall("write_file", map[string]any{"path": "api.py", "content": "corrected"}), "Finished correction",
	)
	store.messages = []modelapi.Message{
		{Role: "user", Content: "Use a simple API"},
		{Role: "assistant", Content: "PRIVATE_PARENT_ANSWER", Reasoning: "PRIVATE_PARENT_THINKING"},
	}
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		answer := map[string]string{
			"plan": "INITIAL_PLAN", "critique": "PLAN_CRITIQUE", "refine": "IMPROVED_PLAN",
			"review": "- api.py: requested behavior is missing. Add the handler.",
		}[phase]
		model.replies[phase] = []string{nativeCall("inspect_fixture", map[string]any{"path": "api.py"}), answer}
	}
	result, err := r.Run(context.Background(), "Implement api.py", approve, emit)
	if err != nil || result.Status != runrecord.Answered || result.Text != "Finished correction" || tools.executed != 2 {
		t.Fatalf("LAT repair: result=%+v writes=%d err=%v", result, tools.executed, err)
	}
	if model.created != 5 || model.released != 5 {
		t.Fatalf("generator scopes not released exactly once: created=%d released=%d", model.created, model.released)
	}
	for _, phase := range []string{"main", "plan", "critique", "refine", "review"} {
		if model.scopes[phase] != 1 {
			t.Errorf("%s used %d generator scopes", phase, model.scopes[phase])
		}
	}
	wantLifecycle := []string{
		"open", "read:inspect_fixture", "close", "open", "read:inspect_fixture", "close",
		"open", "read:inspect_fixture", "close", "main:write_file", "open", "read:inspect_fixture", "close", "main:write_file",
	}
	if !slices.Equal(tools.lifecycle, wantLifecycle) {
		t.Fatalf("read-only lifetimes crossed parent work: %v", tools.lifecycle)
	}
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		requests := model.requests[phase]
		if len(requests) != 2 || !hasNativePair(requests[1].Messages) {
			t.Errorf("%s lost native call/result: %+v", phase, requests)
			continue
		}
		if len(requests[0].Tools) != 1 || requests[0].Tools[0].Name != "inspect_fixture" {
			t.Errorf("%s did not receive the read-only definitions: %+v", phase, requests[0].Tools)
		}
		wantTemp, wantOutput := 1.0, 0
		if phase == "critique" || phase == "review" {
			wantTemp, wantOutput = 0.2, 4096
		}
		if requests[0].Temperature == nil || *requests[0].Temperature != wantTemp || requests[0].MaxOutput != wantOutput {
			t.Errorf("%s used incorrect role sampling: %+v", phase, requests[0])
		}
	}
	assertContains(t, model.requests["critique"][0], "INITIAL_PLAN", "Implement api.py")
	assertContains(t, model.requests["refine"][0], "INITIAL_PLAN", "PLAN_CRITIQUE", "Implement api.py")
	assertContains(t, model.requests["main"][0], "IMPROVED_PLAN", "Implement api.py")
	assertContains(t, model.requests["review"][0], "Use a simple API", "Implement api.py", "Initial draft", r.ProjectInstructions, "IMPROVED_PLAN")
	planCopies := 0
	for _, message := range model.requests["review"][0].Messages {
		planCopies += strings.Count(message.Content, "IMPROVED_PLAN")
	}
	if planCopies != 1 {
		t.Fatalf("review received the refined plan %d times", planCopies)
	}
	assertContains(t, model.requests["main"][2], "requested behavior is missing")
	assertAbsent(t, model.requests["main"][0], "INITIAL_PLAN", "PLAN_CRITIQUE")
	assertAbsent(t, model.requests["review"][0], "PRIVATE_PARENT_ANSWER", "PRIVATE_PARENT_THINKING")
	users := 0
	for _, message := range store.messages {
		if message.Role == "user" && message.Content == "Implement api.py" {
			users++
		}
		if strings.Contains(message.Content, "INITIAL_PLAN") || strings.Contains(message.Content, "PLAN_CRITIQUE") {
			t.Fatal("a private phase transcript entered parent history")
		}
	}
	if users != 1 {
		t.Fatalf("original user request was persisted %d times", users)
	}
	phases := map[string]string{}
	for _, event := range store.events {
		if event.LAT == nil {
			continue
		}
		if !event.Internal {
			t.Fatal("private phase event was published to the parent timeline")
		}
		phases[event.LAT.Phase] = event.LAT.ID
	}
	ids := map[string]bool{}
	for _, phase := range []string{"plan", "critique", "refine", "review"} {
		id := phases[phase]
		if id == "" || ids[id] {
			t.Fatalf("phase audit scopes are missing or shared: %v", phases)
		}
		ids[id] = true
	}
	if len(phases) != len(ids) {
		t.Fatalf("unexpected phase audit scope: %v", phases)
	}
}

func TestLATReviewsAnAnswerWithoutToolWork(t *testing.T) {
	r, model, _, _ := latFixture("A plain answer")
	result, err := r.Run(context.Background(), "Explain the API", approve, emit)
	if err != nil || result.Text != "A plain answer" || len(model.requests["review"]) != 1 {
		t.Fatalf("plain answer skipped review: %+v calls=%v err=%v", result, model.scopes, err)
	}
}

func TestLATReadOnlyPhaseCannotInvokeParentWrites(t *testing.T) {
	r, model, tools, _ := latFixture("Completed answer")
	model.replies["plan"] = []string{
		nativeCall("write_file", map[string]any{"path": "a", "content": "should not exist"}), "INITIAL_PLAN",
	}
	result, err := r.Run(context.Background(), "Explain", approve, emit)
	if err != nil || result.Text != "Completed answer" || tools.executed != 0 || len(tools.views) != 4 {
		t.Fatalf("read-only write attempt: result=%+v main writes=%d views=%d err=%v", result, tools.executed, len(tools.views), err)
	}
	if tools.views[0].executor.prepared != 0 || len(model.requests["plan"]) != 2 || !hasNativePair(model.requests["plan"][1].Messages) {
		t.Fatalf("rejected write was executed or lacked native feedback: %+v", model.requests["plan"])
	}
}

func TestLATPreparationFailureIsAdvisory(t *testing.T) {
	for _, unavailableView := range []bool{false, true} {
		r, model, tools, store := latFixture("Completed answer")
		if unavailableView {
			tools.failure = errors.New("read-only view unavailable")
		} else {
			model.handlers["critique"] = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
				return errors.New("plan critique unavailable")
			}
		}
		result, err := r.Run(context.Background(), "Explain", approve, emit)
		if err != nil || result.Text != "Completed answer" || len(model.requests["refine"]) != 0 {
			t.Fatalf("preparation failure stopped parent: %+v phases=%v err=%v", result, model.scopes, err)
		}
		assertAbsent(t, model.requests["main"][0], "INITIAL_PLAN", "IMPROVED_PLAN")
		found := false
		for _, event := range store.events {
			found = found || event.Kind == "lat_unavailable" && strings.Contains(event.Text, "unavailable")
		}
		if !found {
			t.Fatal("preparation failure was hidden")
		}
	}
}

func TestLATReviewFailureIsAdvisoryAndStorageFailureIsNot(t *testing.T) {
	for _, storageFailure := range []bool{false, true} {
		r, model, _, store := latFixture("Completed answer")
		model.handlers["review"] = func(context.Context, modelapi.Request, func(modelapi.Event) error) error {
			return errors.New("review service unavailable")
		}
		if storageFailure {
			store.failKind = "lat_unavailable"
		}
		result, err := r.Run(context.Background(), "Explain", approve, emit)
		if storageFailure {
			if !errors.Is(err, errRuntimeRecord) {
				t.Fatal("lost durable failure", err)
			}
			continue
		}
		if err != nil || result.Text != "Completed answer" {
			t.Fatalf("review became a completion gate: %+v %v", result, err)
		}
		found := false
		for _, event := range store.events {
			if event.Kind == "lat_clean" {
				t.Fatal("failed review reported clean")
			}
			found = found || event.Kind == "lat_unavailable" && strings.Contains(event.Text, "review service unavailable")
		}
		if !found {
			t.Fatal("review failure was hidden")
		}
	}
}

func TestLATPhaseJournalFailureStopsParent(t *testing.T) {
	r, model, tools, store := latFixture("should not run")
	store.failKind = "lat_message"
	_, err := r.Run(context.Background(), "request", approve, emit)
	if !errors.Is(err, errRuntimeRecord) || len(model.requests["main"]) != 0 || tools.executed != 0 || model.created != model.released {
		t.Fatalf("private audit failure did not stop work: scopes=%d/%d err=%v", model.created, model.released, err)
	}
}

func hasNativePair(messages []modelapi.Message) bool {
	for i, message := range messages {
		if len(message.Calls) == 1 && i+1 < len(messages) && messages[i+1].Role == "tool" && messages[i+1].ToolCallID == message.Calls[0].ID {
			return true
		}
	}
	return false
}

func assertContains(t *testing.T, request modelapi.Request, values ...string) {
	t.Helper()
	var text strings.Builder
	for _, message := range request.Messages {
		text.WriteString(message.Content)
		text.WriteString(message.Reasoning)
	}
	for _, value := range values {
		if !strings.Contains(text.String(), value) {
			t.Errorf("request lost %q", value)
		}
	}
}

func assertAbsent(t *testing.T, request modelapi.Request, values ...string) {
	t.Helper()
	for _, message := range request.Messages {
		for _, value := range values {
			if strings.Contains(message.Content, value) || strings.Contains(message.Reasoning, value) {
				t.Errorf("request inherited %q", value)
			}
		}
	}
}

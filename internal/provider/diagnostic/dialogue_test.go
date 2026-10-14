// Purpose: Verify PFC setup probes use runtime grammar, bounded correction and actionable failures.
// Pattern: Generator contract tests.
// Usage: go test ./internal/provider/diagnostic -run TestDialogue.
// Related: internal/provider/diagnostic/check.go, internal/agent/pfc/pfc.go.
package diagnostic

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func TestDialogueAcceptsRuntimeDecisionWhitespace(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		reply := " { \"tool\" : \"read_file\" }\n"
		if calls > 0 {
			reply = "AGENTS.md\r\n"
		}
		calls++
		return emit(modelapi.Event{Delta: reply, Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", false, false, nil)
	if err != nil || !caps.ToolDialogue || calls != 2 {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
}

func TestDialogueCorrectsOneInvalidField(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		reply := `{"tool":"read_file"}`
		if calls == 2 {
			reply = `{"path":"AGENTS.md"}`
		} else if calls == 3 {
			last := r.Messages[len(r.Messages)-1].Content
			if !strings.Contains(last, "literal path") || !strings.Contains(last, "AGENTS.md") {
				t.Fatalf("missing corrective feedback: %q", last)
			}
			reply = "AGENTS.md"
		}
		return emit(modelapi.Event{Delta: reply, Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", false, false, nil)
	if err != nil || !caps.ToolDialogue || calls != 3 {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
}

func TestDialogueExplainsFailedStageAfterCorrection(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		return emit(modelapi.Event{Delta: "I cannot access files.", Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", false, false, nil)
	if err == nil || caps.ToolDialogue || calls != 2 {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
	for _, want := range []string{"PFC", "decision", "read_file", "I cannot access files."} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("missing %q in %v", want, err)
		}
	}
}

func TestDialogueRejectsWrappedPathAfterCorrection(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		reply := `{"path":"AGENTS.md"}`
		if calls == 1 {
			reply = `{"tool":"read_file"}`
		}
		return emit(modelapi.Event{Delta: reply, Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", false, false, nil)
	if err == nil || caps.ToolDialogue || calls != 3 || !strings.Contains(err.Error(), "read_file.path") || !strings.Contains(err.Error(), "not a JSON wrapper") {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
}

func TestDialogueTransportRetryDiscardsPartialAttempt(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, r modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		if calls == 1 {
			if err := emit(modelapi.Event{Delta: "stale", Thinking: "stale reasoning"}); err != nil {
				return err
			}
			if err := emit(modelapi.Event{Retry: &modelapi.Retry{Attempt: 2}}); err != nil {
				return err
			}
			return emit(modelapi.Event{Delta: `{"tool":"read_file"}`, Done: true})
		}
		if calls == 2 {
			return emit(modelapi.Event{Delta: "AGENTS.md", Done: true})
		}
		return emit(modelapi.Event{Delta: "391", Done: true})
	})
	caps, err := Check(context.Background(), generate, "alias", true, false, nil)
	if err != nil || !caps.ToolDialogue || calls != 3 {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
}

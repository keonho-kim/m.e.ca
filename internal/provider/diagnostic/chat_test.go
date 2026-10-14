// Purpose: Verify connection checks accept ordinary answers without invoking the PFC protocol.
// Pattern: Generator contract tests.
// Usage: go test ./internal/provider/diagnostic -run TestChat.
// Related: internal/provider/diagnostic/check.go, internal/provider/diagnostic/chat.go.
package diagnostic

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
)

func TestChatConnectionAcceptsOrdinaryResponse(t *testing.T) {
	calls := 0
	generate := generatorFunc(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		calls++
		for _, message := range request.Messages {
			for _, forbidden := range []string{"read_file", "AGENTS.md", "ARGUMENT", "PFC"} {
				if strings.Contains(message.Content, forbidden) {
					t.Errorf("connection requested tool dialogue: %q", message.Content)
				}
			}
		}
		return emit(modelapi.Event{Delta: "\n\n안녕하세요. 무엇을 도와드릴까요?", Done: true, Streaming: true})
	})
	caps, err := Check(context.Background(), generate, "alias", false, false, nil)
	if err != nil || !caps.Chat || !caps.Streaming || calls != 1 {
		t.Fatalf("caps=%+v calls=%d err=%v", caps, calls, err)
	}
}

func TestChatConnectionRejectsMissingAnswerOrCompletion(t *testing.T) {
	for _, event := range []modelapi.Event{{Delta: " \n", Done: true}, {Delta: "hello"}, {Thinking: "reasoning only", Done: true}} {
		generate := generatorFunc(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
			return emit(event)
		})
		if _, err := Check(context.Background(), generate, "alias", false, false, nil); err == nil {
			t.Fatalf("accepted incomplete response: %+v", event)
		}
	}
}

func TestChatConnectionRetryDiscardsPreviousAnswer(t *testing.T) {
	generate := generatorFunc(func(_ context.Context, _ modelapi.Request, emit func(modelapi.Event) error) error {
		for _, event := range []modelapi.Event{{Delta: "old answer", Done: true}, {Retry: &modelapi.Retry{Attempt: 2}}, {Done: true}} {
			if err := emit(event); err != nil {
				return err
			}
		}
		return nil
	})
	if _, err := Check(context.Background(), generate, "alias", false, false, nil); err == nil {
		t.Fatal("failed attempt's answer satisfied the retry")
	}
}

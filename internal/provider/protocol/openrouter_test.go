// Purpose: Verify ordered reasoning-state round trips without exposing encrypted data or duplicating text.
// Pattern: Provider protocol contract tests.
// Usage: go test ./internal/provider/protocol -run TestOpenRouter.
// Related: internal/provider/protocol/openrouter.go, internal/provider/protocol/chunk.go.
package protocol

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestOpenRouterReasoningState(t *testing.T) {
	blocks := []string{
		`{"type":"reasoning.text","text":"brief","index":0,"signature":null,"format":"anthropic-claude-v1"}`,
		`{"type":"reasoning.text","text":" reasoning","index":0,"signature":"signed","format":"anthropic-claude-v1"}`,
		`{"type":"reasoning.encrypted","data":"opaque-state","index":1,"future":{"keep":true}}`,
	}
	stream := ""
	for _, block := range blocks {
		stream += `data: {"choices":[{"delta":{"reasoning_details":[` + block + `]}}]}` + "\n\n"
	}
	stream += "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n"
	response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}
	defer response.Body.Close()
	var state []byte
	var thinking string
	err := ReadResponse(context.Background(), response, connection.OpenRouter, func(event modelapi.Event) error {
		thinking += event.Thinking
		if event.Done {
			state = event.ReasoningState
		}
		return nil
	})
	if err != nil || thinking != "brief reasoning" || len(state) == 0 {
		t.Fatalf("thinking=%q state=%s err=%v", thinking, state, err)
	}
	for _, kind := range []connection.ProviderKind{connection.OpenRouter, connection.VLLM} {
		messages, err := Messages(kind, []modelapi.Message{{Role: "assistant", Content: "done", Reasoning: thinking, ReasoningState: state}})
		if err != nil {
			t.Fatal(err)
		}
		if kind != connection.OpenRouter {
			if len(messages[0].ReasoningDetails) != 0 {
				t.Fatal("OpenRouter state leaked to another provider")
			}
			continue
		}
		if len(messages[0].ReasoningDetails) != 2 || messages[0].Reasoning != "" {
			t.Fatalf("reasoning duplicated or blocks lost: %+v", messages[0])
		}
		var detail struct{ Text, Signature string }
		if err := json.Unmarshal(messages[0].ReasoningDetails[0], &detail); err != nil {
			t.Fatal(err)
		}
		if detail.Text != "brief reasoning" || detail.Signature != "signed" || string(messages[0].ReasoningDetails[1]) != blocks[2] {
			t.Fatalf("fragments or encrypted block changed: %+v", messages[0])
		}
	}
}

func TestOpenRouterReasoningBlockBoundaries(t *testing.T) {
	state := streamState{}
	blocks := []json.RawMessage{
		json.RawMessage(`{"type":"reasoning.summary","summary":"one","index":0}`),
		json.RawMessage(`{"type":"reasoning.summary","summary":" two","index":0}`),
		json.RawMessage(`{"type":"reasoning.encrypted","data":"a","index":0}`),
		json.RawMessage(`{"type":"reasoning.encrypted","data":"b","index":0}`),
		json.RawMessage(`{"type":"reasoning.text","text":"signed","signature":"sig","index":0}`),
		json.RawMessage(`{"type":"reasoning.text","text":"another","index":0}`),
		json.RawMessage(`{"type":"reasoning.text","signature":"sig2","index":0}`),
	}
	if _, err := state.routerDetails(blocks, true); err != nil {
		t.Fatal(err)
	}
	body, err := routerContinuation(state.reasoningDetails, true)
	if err != nil {
		t.Fatal(err)
	}
	var completed []json.RawMessage
	if err := json.Unmarshal(body, &completed); err != nil {
		t.Fatal(err)
	}
	if len(completed) != 5 {
		t.Fatalf("block boundaries lost: %s", completed)
	}
	var first struct{ Summary string }
	var last struct{ Text, Signature string }
	if err := json.Unmarshal(completed[0], &first); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(completed[4], &last); err != nil {
		t.Fatal(err)
	}
	if first.Summary != "one two" || last.Text != "another" || last.Signature != "sig2" {
		t.Fatalf("fragments not assembled: %+v %+v", first, last)
	}
	history := `[{"type":"reasoning.summary","summary":"first"},{"type":"reasoning.summary","summary":"second"}]`
	if details, err := routerHistory([]byte(history)); err != nil || len(details) != 2 {
		t.Fatalf("already completed history was merged: %s %v", details, err)
	}
}

func TestOpenRouterReasoningTextIsNotDuplicated(t *testing.T) {
	state := streamState{provider: connection.OpenRouter}
	var thinking string
	err := state.consume([]byte(`{"choices":[{"message":{"content":"done","reasoning":"brief","reasoning_details":[{"type":"reasoning.summary","summary":"brief"}]},"finish_reason":"stop"}]}`), true, func(e modelapi.Event) error { thinking += e.Thinking; return nil })
	if err != nil || thinking != "brief" {
		t.Fatalf("thinking=%q err=%v", thinking, err)
	}
	if err := state.complete(func(e modelapi.Event) error {
		if !json.Valid(e.ReasoningState) {
			t.Fatal("non-streaming continuation lost")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestOpenRouterRejectsInvalidReasoningState(t *testing.T) {
	for _, content := range []string{`null`, `{}`, `[null]`, `[{}]`, `[1]`} {
		if _, err := Messages(connection.OpenRouter, []modelapi.Message{{Role: "assistant", ReasoningState: []byte(content)}}); err == nil {
			t.Fatalf("invalid continuation accepted: %s", content)
		}
	}
}

func TestOpenRouterCancellationDoesNotPublishContinuation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	stream := "data: {\"choices\":[{\"delta\":{\"reasoning_details\":[{\"type\":\"reasoning.text\",\"text\":\"partial\"}]}}]}\n\ndata: [DONE]\n\n"
	response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader(stream))}
	defer response.Body.Close()
	var continuation []byte
	done := false
	err := ReadResponse(ctx, response, connection.OpenRouter, func(event modelapi.Event) error {
		done = done || event.Done
		continuation = append(continuation, event.ReasoningState...)
		cancel()
		return nil
	})
	if !errors.Is(err, context.Canceled) || done || len(continuation) != 0 {
		t.Fatalf("cancelled state published: done=%t state=%s err=%v", done, continuation, err)
	}
}

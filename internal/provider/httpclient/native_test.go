// Purpose: Verify native call retries and exact tokenization include the generation tool contract.
// Pattern: HTTP integration contract tests.
// Usage: go test ./internal/provider/httpclient -run TestNative.
// Related: internal/provider/httpclient/context.go, internal/provider/protocol/tool_calls.go.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/protocol"
	"net/http"
	"sync/atomic"
	"testing"
)

func TestNativeRetryDiscardsPartialArguments(t *testing.T) {
	var attempts atomic.Int32
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if attempts.Add(1) == 1 {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"old\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\"}}]}}]}\n\n")
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"new\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\\\"a\\\"}\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\ndata: [DONE]\n\n")
	})
	var calls []toolapi.Call
	retries := 0
	err := client.Generate(context.Background(), recoveryRequest(), func(event modelapi.Event) error {
		if event.Retry != nil {
			retries++
		}
		calls = append(calls, event.Calls...)
		return nil
	})
	if err != nil || retries != 1 || len(calls) != 1 || calls[0].ID != "new" || string(calls[0].Arguments) != `{"path":"a"}` {
		t.Fatalf("calls=%+v retries=%d: %v", calls, retries, err)
	}
}

func TestNativeTokenizationUsesGenerationSchemas(t *testing.T) {
	request := recoveryRequest()
	request.Tools = []toolapi.Definition{{Name: "read_file", Description: "Read", Parameters: []toolapi.Parameter{{Name: "path", Kind: toolapi.LiteralPath, Required: true, MaxBytes: 4096}}}}
	expected := protocol.Tools(request.Tools)
	var tokenized, generated atomic.Bool
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			fmt.Fprint(w, `{"data":[{"id":"alias","max_model_len":100}]}`)
			return
		}
		var body struct{ Tools []protocol.Tool }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		// Compare serialized schemas, including numeric bounds.
		actual, err := json.Marshal(body.Tools)
		want, wantErr := json.Marshal(expected)
		if err != nil || wantErr != nil || !bytes.Equal(actual, want) {
			t.Errorf("tools=%s want=%s", actual, want)
		}
		if r.URL.Path == "/tokenize" {
			tokenized.Store(true)
			fmt.Fprint(w, `{"count":10,"max_model_len":100}`)
			return
		}
		generated.Store(true)
		fmt.Fprint(w, `{"choices":[{"message":{"content":"hello"},"finish_reason":"stop"}]}`)
	})
	if _, err := prepareModel(client, context.Background(), "alias"); err != nil {
		t.Fatal(err)
	}
	if err := client.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if !tokenized.Load() || !generated.Load() {
		t.Fatal("missing tokenization or generation")
	}
}

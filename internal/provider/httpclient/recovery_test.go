// Purpose: Verify large streams, retry reset events and explicit provider errors after recovery attempts.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestRecovery.
// Related: internal/provider/transport/retry.go, internal/provider/protocol/stream.go.
package httpclient

import (
	"context"
	"errors"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

func recoveryRequest() modelapi.Request {
	return modelapi.Request{ModelID: "alias", MaxOutput: 32, Messages: []modelapi.Message{{Role: "user", Content: "hello"}}}
}

func TestRecoveryStreamHasNoAggregateTransportLimit(t *testing.T) {
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		line := ":" + strings.Repeat("heartbeat", 1000) + "\n\n"
		for range 1100 {
			fmt.Fprint(w, line)
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\ndata: [DONE]\n\n")
	})
	var answer string
	err := client.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error { answer += e.Delta; return nil })
	if err != nil || answer != "hello" {
		t.Fatalf("answer=%q err=%v", answer, err)
	}
}

func TestRecoveryStreamAcceptsLargeFrame(t *testing.T) {
	reply := strings.Repeat("x", 2*1024*1024)
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\ndata: [DONE]\n\n", reply)
	})
	size := 0
	err := client.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error {
		if e.Retry != nil {
			t.Error("valid large frame retried")
		}
		size += len(e.Delta)
		return nil
	})
	if err != nil || size != len(reply) {
		t.Fatalf("size=%d err=%v", size, err)
	}
}

func TestRecoveryPartialStreamRestartsWithoutConcatenating(t *testing.T) {
	var calls atomic.Int32
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if calls.Add(1) == 1 {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"unfinished\"}}]}\n\n")
			return
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"recovered\"}}]}\n\ndata: [DONE]\n\n")
	})
	answer, retries := "", 0
	err := client.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error {
		if e.Retry != nil {
			answer = ""
			retries++
		}
		answer += e.Delta
		return nil
	})
	if err != nil || answer != "recovered" || retries != 1 || calls.Load() != 2 {
		t.Fatalf("answer=%q retries=%d calls=%d err=%v", answer, retries, calls.Load(), err)
	}
}

func TestRecoveryExhaustionReturnsSanitizedCause(t *testing.T) {
	var calls atomic.Int32
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error":{"code":"invalid_api_key","message":"bad credential test-secret"}}`)
	})
	err := client.Generate(context.Background(), recoveryRequest(), func(modelapi.Event) error { return nil })
	if err == nil || calls.Load() != 3 || !strings.Contains(err.Error(), "3 attempts") || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "invalid_api_key") || strings.Contains(err.Error(), "test-secret") {
		t.Fatalf("calls=%d err=%v", calls.Load(), err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := client.Generate(ctx, recoveryRequest(), func(modelapi.Event) error { return nil }); !errors.Is(err, context.Canceled) || calls.Load() != 3 {
		t.Fatalf("cancel retried: calls=%d err=%v", calls.Load(), err)
	}
}

func TestRecoveryReadsUsageAfterOutputLimit(t *testing.T) {
	client := mockClient(t, connection.LiteLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":\"length\"}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":12,\"completion_tokens\":32}}\n\ndata: [DONE]\n\n")
	})
	var latest *modelapi.TokenUsage
	retries := 0
	err := client.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error {
		if e.Usage != nil {
			latest = e.Usage
		}
		if e.Retry != nil {
			if latest == nil || latest.Output != 32 {
				t.Error("retry began before final usage was received")
			}
			latest = nil
			retries++
		}
		return nil
	})
	if err == nil || retries != 2 || latest == nil || latest.Output != 32 || !strings.Contains(err.Error(), "output token limit") {
		t.Fatalf("usage=%+v retries=%d err=%v", latest, retries, err)
	}
}

func TestRecoveryMetadataUsesFreshDecode(t *testing.T) {
	var calls atomic.Int32
	client := mockClient(t, connection.SGLang, func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			fmt.Fprint(w, `{"model_path":"Qwen/Qwen3.8-27B","served_model_name":42}`)
			return
		}
		fmt.Fprint(w, `{"served_model_name":"alias","model_type":"qwen"}`)
	})
	report, err := catalogFor(client).Identify(context.Background(), "alias")
	if err != nil || report.Unavailable || len(report.Originals()) != 0 || calls.Load() != 2 {
		t.Fatalf("report=%+v calls=%d err=%v", report, calls.Load(), err)
	}
}

func TestRecoveryStreamHasNoAggregateTextLimit(t *testing.T) {
	client := mockClient(t, connection.VLLM, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		text := strings.Repeat("x", 128*1024)
		for range 40 {
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"%s\"}}]}\n\n", text)
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	})
	bytes := 0
	err := client.Generate(context.Background(), recoveryRequest(), func(e modelapi.Event) error { bytes += len(e.Delta); return nil })
	if err != nil || bytes != 5*1024*1024 {
		t.Fatalf("decoded bytes=%d err=%v", bytes, err)
	}
}

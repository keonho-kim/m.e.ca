// Purpose: Verify live compatible deltas, disconnect ownership and honest retry reset failures.
// Pattern: Real HTTP streaming tests with channel-controlled model completion.
// Usage: go test -race ./internal/serverapi -run 'TestOpenAIStream|TestOpenAIDisconnect'.
// Related: internal/serverapi/openai/stream.go, internal/serverapi/openai/output.go.
package serverapi

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
)

type compatibleFrame struct {
	Type    string
	Delta   string
	Choices []struct{ Delta struct{ Content string } }
	Error   *struct{ Code string }
	Code    string
}

func nextCompatibleFrame(t *testing.T, reader *bufio.Reader) compatibleFrame {
	t.Helper()
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, "data: "))
		if body == "[DONE]" {
			return compatibleFrame{Type: "done"}
		}
		var frame compatibleFrame
		if err := json.Unmarshal([]byte(body), &frame); err != nil {
			t.Fatal(body, err)
		}
		return frame
	}
}
func compatibleStream(t *testing.T, f *fixture, path, body string) (*http.Response, *bufio.Reader) {
	t.Helper()
	transport := openAITransport(t, f, "sse")
	srv := httptest.NewServer(transport)
	t.Cleanup(func() { srv.CloseClientConnections(); srv.Close() })
	req, err := http.NewRequestWithContext(deadline(t), "POST", srv.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "localhost:8787"
	req.Header.Set("Authorization", "Bearer "+testToken)
	req.Header.Set("Content-Type", "application/json")
	reply, err := srv.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = reply.Body.Close() })
	if reply.StatusCode != 200 {
		data, _ := io.ReadAll(reply.Body)
		t.Fatal(reply.StatusCode, string(data))
	}
	return reply, bufio.NewReader(reply.Body)
}

func TestOpenAIDisconnectKeepsAcceptedTaskRunning(t *testing.T) {
	release := make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: "live output", Streaming: true}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-release:
		}
		return emit(model.Event{Delta: " finished", Done: true, Streaming: true})
	}), false)
	reply, reader := compatibleStream(t, f, "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"hello"}],"stream":true}`)
	for {
		frame := nextCompatibleFrame(t, reader)
		if len(frame.Choices) > 0 && frame.Choices[0].Delta.Content != "" {
			break
		}
	}
	if f.app.State().Task.Status != "running" {
		t.Fatal("stream arrived after task completion")
	}
	_ = reply.Body.Close()
	if f.app.State().Task.Status != "running" {
		t.Fatal("observer disconnect stopped task")
	}
	cursor := f.app.State().Cursor
	close(release)
	if task := waitFinished(t, deadline(t), f.app, cursor); task.Status != "answered" {
		t.Fatal(task)
	}
}

func TestOpenAIStreamRetryErrorsWithoutCancellingTask(t *testing.T) {
	retry, finish := make(chan struct{}), make(chan struct{})
	f := newFixture(t, generateFunc(func(ctx context.Context, _ model.Request, emit func(model.Event) error) error {
		if err := emit(model.Event{Delta: "old output", Streaming: true}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-retry:
		}
		if err := emit(model.Event{Retry: &model.Retry{Attempt: 2, MaxAttempts: 2, Reason: "fixture"}}); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-finish:
		}
		return emit(model.Event{Delta: "new output", Done: true, Streaming: true})
	}), false)
	reply, reader := compatibleStream(t, f, "/v1/responses", `{"model":"meca","input":"hello","stream":true}`)
	id := reply.Header.Get("X-Request-ID")
	for {
		if frame := nextCompatibleFrame(t, reader); frame.Type == "response.output_text.delta" {
			break
		}
	}
	close(retry)
	for {
		frame := nextCompatibleFrame(t, reader)
		if frame.Type == "error" {
			if frame.Code != "stream_rewrite" {
				t.Fatal(frame)
			}
			break
		}
	}
	if f.app.State().Task.Status != "running" {
		t.Fatal("stream rewrite stopped engine")
	}
	cursor := f.app.State().Cursor
	close(finish)
	waitFinished(t, deadline(t), f.app, cursor)
	lookup := openAIRequest(t, openAITransport(t, f, "socket"), "GET", "/v1/responses/"+id, "", "")
	if lookup.Code != 200 || !strings.Contains(lookup.Body.String(), `"text":"new output"`) || strings.Contains(lookup.Body.String(), "old output") {
		t.Fatal(lookup.Code, lookup.Body.String())
	}
}

func TestOpenAIStreamReturnsWholeOriginalBeyondPreviewLimit(t *testing.T) {
	text := strings.Repeat("원문 ", 100000)
	f := newFixture(t, generateFunc(func(_ context.Context, _ model.Request, emit func(model.Event) error) error {
		return emit(model.Event{Delta: text, Done: true, Streaming: true, Usage: &model.TokenUsage{Input: 12, Output: 8}})
	}), false)
	transport := openAITransport(t, f, "sse")
	for _, kind := range []string{"chat", "response"} {
		path, body := "/v1/chat/completions", `{"model":"meca","messages":[{"role":"user","content":"hello"}],"stream":true,"stream_options":{"include_usage":true}}`
		if kind == "response" {
			path, body = "/v1/responses", `{"model":"meca","input":"hello","stream":true}`
		}
		reply := openAIRequest(t, transport, "POST", path, body, "")
		if reply.Code != 200 || strings.Contains(reply.Body.String(), `"error":{`) {
			t.Fatal(reply.Code, reply.Body.String()[:min(1000, reply.Body.Len())])
		}
		reader := bufio.NewReader(strings.NewReader(reply.Body.String()))
		var output strings.Builder
		finished := false
		for !finished {
			frame := nextCompatibleFrame(t, reader)
			if kind == "chat" {
				if len(frame.Choices) > 0 {
					output.WriteString(frame.Choices[0].Delta.Content)
				}
				finished = frame.Type == "done"
			} else {
				if frame.Type == "response.output_text.delta" {
					output.WriteString(frame.Delta)
				}
				finished = frame.Type == "response.completed"
			}
		}
		if output.String() != strings.TrimSpace(text) {
			t.Fatalf("%s streamed %d bytes, expected %d", kind, output.Len(), len(strings.TrimSpace(text)))
		}
		if !strings.Contains(reply.Body.String(), `"total_tokens":20`) {
			t.Fatal("measured usage missing")
		}
	}
}

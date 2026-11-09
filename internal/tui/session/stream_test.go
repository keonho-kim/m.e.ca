// Purpose: Verify real SSE updates a terminal response before HTTP completion and commits it once.
// Pattern: HTTP-to-terminal integration test.
// Usage: go test ./internal/tui/session -run TestStreamingResponse.
// Related: internal/provider/protocol/stream.go, internal/agent/run/generate.go, internal/tui/session/events.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

func TestStreamingResponseBeforeServerCompletes(t *testing.T) {
	a := fixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	a.ctx = ctx
	resume := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Messages []modelapi.Message `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"first\",\"reasoning\":\"inspect \"}}]}\n\n")
		if err := http.NewResponseController(w).Flush(); err != nil {
			t.Error(err)
			return
		}
		select {
		case <-resume:
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" reply\",\"reasoning\":\"more\"}}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":123,\"completion_tokens\":6}}\n\ndata: [DONE]\n\n")
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.connection.client, err = httpclient.NewClient(connection, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.connection.client.Close)
	t.Cleanup(func() { cancel(); a.workers.Wait() })
	a.connection.selection = modelconn.Selection{Connection: connection, Model: "streaming-model"}
	a.chat.historyEnd = true
	a.chat.output.Append(strings.Repeat("earlier history\n", 40), false)
	a.scroll(-5, 80, a.chatHeight(80, a.bodyHeight(24)))
	if a.chat.output.Follow {
		t.Fatal("history inspection did not pause following")
	}
	a.chat.output.Append((transcript.Formatter{}).User("hello"), false)
	a.startRequest(composer.Capture("hello", a.selectedPaths()))
	if !a.chat.output.Follow {
		t.Fatal("a new request did not resume following")
	}
	for a.chat.stream != "first" {
		select {
		case n := <-a.events:
			a.receive(n)
			if n.Done {
				t.Fatalf("task ended before partial answer: %v", n.Err)
			}
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		}
	}
	if !a.work.state.active() {
		t.Fatal("partial answer arrived only after completion")
	}
	a.render()
	var visible strings.Builder
	for y := 0; y < 24; y++ {
		visible.WriteString(screenLine(a.screen, y, 80) + "\n")
	}
	if !strings.Contains(visible.String(), "first") || strings.Contains(visible.String(), "DONE") {
		t.Fatalf("partial not visible or protocol exposed: %s", visible.String())
	}
	close(resume)
	drainTask(t, a, false)
	var transcript strings.Builder
	for _, block := range a.chat.output.Blocks {
		transcript.WriteString(block.Text + "\n")
	}
	if a.chat.stream != "" || a.chat.thinking != "" || strings.Count(transcript.String(), "first reply") != 1 || strings.Count(transcript.String(), "inspect more") != 1 {
		t.Fatalf("duplicated/unfinished stream: %q", transcript.String())
	}
	summary := a.chat.output.Blocks[len(a.chat.output.Blocks)-1]
	// Fast streams can finish below the persisted millisecond precision.
	// Summary tests cover TPS with an explicit generation duration.
	if !summary.Centered || !summary.Thinking || !strings.Contains(summary.Text, "입력 123 | 출력 6") {
		t.Fatalf("final request summary missing: %+v", summary)
	}
	messages, err := a.opts.Store.ContextMessages(ctx, a.opts.Session.ID, 0, 16*1024*1024, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range messages {
		if strings.Contains(message.Content, "작업 시간") || strings.Contains(message.Content, "Worked for") {
			t.Fatal("UI measurements leaked into model context")
		}
	}
	a.chat.output = viewport.View{Follow: true}
	a.chat.historyCursor = 0
	if err := a.loadHistory(); err != nil {
		t.Fatal(err)
	}
	restored := a.chat.output.Blocks[len(a.chat.output.Blocks)-1]
	if restored.Text != summary.Text || !restored.Centered || !restored.Thinking {
		t.Fatalf("restored measurements changed: %+v", restored)
	}
	if tokens, err := a.opts.Store.TokenUsage(ctx, a.opts.Session.ID); err != nil || tokens.Input != 123 || tokens.Output != 6 {
		t.Fatalf("request summary duplicated usage: %+v, %v", tokens, err)
	}
}

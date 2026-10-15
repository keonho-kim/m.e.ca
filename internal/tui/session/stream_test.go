// Purpose: Verify real SSE updates a terminal response before HTTP completion and commits it once.
// Pattern: HTTP-to-terminal integration test.
// Usage: go test ./internal/tui/session -run TestStreamingResponse.
// Related: internal/provider/protocol/stream.go, internal/agent/run/generate.go, internal/tui/session/events.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/httpclient"
	"github.com/keonho-kim/v-code/internal/tui/transcript"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
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
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\" reply\",\"reasoning\":\"more\"}}]}\n\ndata: [DONE]\n\n")
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.OpenAICompatible, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(connection, "", "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.client.Close)
	t.Cleanup(func() { cancel(); a.workers.Wait() })
	a.selection = modelconn.Selection{Connection: connection, Model: "streaming-model"}
	a.output.Append(transcript.User("hello"), false)
	a.startTask("hello")
	for a.stream != "first" {
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
	if !a.busy {
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
	for _, block := range a.output.Blocks {
		transcript.WriteString(block.Text + "\n")
	}
	if a.stream != "" || a.thinking != "" || strings.Count(transcript.String(), "first reply") != 1 || strings.Count(transcript.String(), "inspect more") != 1 {
		t.Fatalf("duplicated/unfinished stream: %q", transcript.String())
	}
}

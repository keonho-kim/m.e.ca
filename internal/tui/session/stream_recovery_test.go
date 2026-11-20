// Purpose: Verify automatic recovery and single-response presentation in a terminal with long retained context.
// Pattern: HTTP-to-terminal integration tests.
// Usage: go test ./internal/tui/session -run TestStreamingResponseDoesNotReplay.
// Related: internal/provider/httpclient/generation.go, internal/agent/run/generate.go, internal/tui/session/stream_test.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestStreamingResponseDoesNotReplay(t *testing.T) {
	const answer = "Answer begins here."
	const continuation = " Continued automatically."
	const tail = "End of thinking."
	thought := strings.Repeat("An earlier line of thinking.\n\n", 260) + tail
	history := "Earlier context.\n" + strings.Repeat("A long, retained conversation.\n", 5000)
	for _, tc := range []struct {
		name, finalFrames, failure string
		recover, correctTool       bool
		output                     int64
	}{
		{
			name: "clean EOF after terminal reason",
			finalFrames: "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
				"data: {\"choices\":[],\"usage\":{\"prompt_tokens\":196000,\"completion_tokens\":240}}\n\n",
			output: 240,
		},
		{name: "interruption after output", recover: true},
		{
			name: "invalid tool call after output",
			finalFrames: "data: {\"choices\":[{\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"invalid\",\"type\":\"function\",\"function\":{\"name\":\"read_file\",\"arguments\":\"{\\\"path\\\":\"}}]},\"finish_reason\":\"tool_calls\"}]}\n\n" +
				"data: [DONE]\n\n",
			recover: true, correctTool: true,
		},
		{
			name:        "output limit",
			finalFrames: "data: {\"choices\":[{\"delta\":{},\"finish_reason\":\"length\"}]}\n\ndata: [DONE]\n\n",
			recover:     true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := fixture(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
			a.ctx = ctx
			t.Cleanup(cancel)
			store, session := a.opts.Store, a.opts.Session.ID
			if err := store.Commit(ctx, session, []runrecord.Entry{
				{Message: &modelapi.Message{Role: "user", Content: history}},
				{Message: &modelapi.Message{Role: "assistant", Content: "Earlier result."}},
			}); err != nil {
				t.Fatal(err)
			}
			if err := a.loadHistory(); err != nil {
				t.Fatal(err)
			}

			var attempts, measurements atomic.Int32
			resume := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct{ Messages []modelapi.Message }
				if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				var hasHistory bool
				for _, message := range request.Messages {
					hasHistory = hasHistory || message.Content == history
				}
				if !hasHistory {
					t.Error("the model request omitted its long retained context")
				}
				if r.URL.Path == "/tokenize" {
					measurements.Add(1)
					w.Header().Set("Content-Type", "application/json")
					fmt.Fprint(w, "{\"count\":196000,\"max_model_len\":262144}")
					return
				}
				w.Header().Set("Content-Type", "text/event-stream")
				if attempts.Add(1) > 1 {
					var correction bool
					for _, message := range request.Messages {
						correction = correction || strings.Contains(message.Content, "Validation error:")
						if strings.Contains(message.Content, "Unfinished assistant draft") {
							t.Error("retry included an abandoned draft")
						}
					}
					if correction != tc.correctTool {
						t.Errorf("correction feedback present=%v, want %v", correction, tc.correctTool)
					}
					middle := len(answer) / 2
					fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q}}]}\n\n", answer[:middle])
					fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning\":%q,\"content\":%q}}]}\n\n", thought, answer[middle:])
					fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":%q},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", continuation)
					return
				}
				fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"reasoning\":%q,\"content\":%q}}]}\n\n", thought, answer)
				if err := http.NewResponseController(w).Flush(); err != nil {
					t.Error(err)
					return
				}
				select {
				case <-resume:
					fmt.Fprint(w, tc.finalFrames)
				case <-r.Context().Done():
				}
			}))
			t.Cleanup(server.Close)
			t.Cleanup(func() { cancel(); a.workers.Wait() })
			conn, err := modelconn.New(modelconn.VLLM, server.URL)
			if err != nil {
				t.Fatal(err)
			}
			client, err := httpclient.NewClient(conn, "")
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(client.Close)
			info := modelconn.ModelInfo{ID: "stream-fixture", ContextWindow: 262144}
			if err := client.BindModel(info); err != nil {
				t.Fatal(err)
			}
			a.connection.client = client
			a.connection.selection = modelconn.Selection{Connection: conn, Model: info.ID, Details: &info}

			a.startRequest(requestinput.Capture("Continue the previous work", nil))
			var visible bool
			var outcome notice
			retries, starts := 0, 0
			for a.work.state.active() {
				select {
				case n := <-a.events:
					a.receive(n)
					if n.Approval != nil || n.Event != nil && n.Event.Kind == "tool_started" {
						t.Fatal("an invalid tool call reached execution")
					}
					if n.Event != nil {
						switch n.Event.Kind {
						case "retry":
							retries++
							if a.chat.stream != answer || a.chat.thinking != thought {
								t.Fatal("retry cleared the current TUI response")
							}
						case "generation_started":
							starts++
						}
					}
					if visible && n.Stream != nil && a.chat.stream != answer && a.chat.stream != answer+continuation {
						t.Fatalf("TUI showed an incomplete replay: %q", a.chat.stream)
					}
					if !visible && a.chat.stream == answer && a.chat.thinking == thought {
						visible = true
						a.render()
						var screen strings.Builder
						for y := 0; y < 24; y++ {
							screen.WriteString(screenLine(a.screen, y, 80) + "\n")
						}
						if strings.Count(screen.String(), answer) != 1 || strings.Count(screen.String(), tail) != 1 {
							t.Fatalf("partial reply missing or duplicated on screen:\n%s", screen.String())
						}
						close(resume)
					}
					if n.Done {
						outcome = n
					}
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				}
			}
			wantAttempts, wantMeasurements := int32(1), int32(1)
			wantAnswer := answer
			if tc.recover {
				wantAttempts++
				wantAnswer += continuation
			}
			if tc.correctTool || tc.name == "output limit" {
				wantMeasurements++
			}
			if !visible || attempts.Load() != wantAttempts || measurements.Load() != wantMeasurements ||
				retries != int(wantAttempts)-1 || starts != 1 || !outcome.Done {
				t.Fatalf("unexpected lifecycle: visible=%v attempts=%d measurements=%d retries=%d drafts=%d done=%v",
					visible, attempts.Load(), measurements.Load(), retries, starts, outcome.Done)
			}
			if tc.failure == "" {
				if outcome.Err != nil || outcome.RunStatus != runrecord.Answered || a.chat.tokenTotal.Output != tc.output {
					t.Fatalf("completed reply lost: status=%s output=%d error=%v", outcome.RunStatus, a.chat.tokenTotal.Output, outcome.Err)
				}
			} else if outcome.Err == nil || !strings.Contains(strings.ToLower(outcome.Err.Error()), tc.failure) || outcome.RunStatus != runrecord.Stopped {
				t.Fatalf("interruption lost: status=%s error=%v", outcome.RunStatus, outcome.Err)
			}

			for _, restore := range []bool{false, true} {
				if restore {
					if err := a.loadHistory(); err != nil {
						t.Fatal(err)
					}
				}
				var transcript strings.Builder
				for _, block := range a.chat.output.Blocks {
					transcript.WriteString(block.Text + "\n")
				}
				text := transcript.String()
				thoughtIndex := strings.Index(text, tail)
				answerIndex := strings.Index(text, answer)
				if strings.Count(text, answer) != 1 || strings.Count(text, tail) != 1 || thoughtIndex > answerIndex {
					t.Fatalf("reply missing, duplicated or reordered (restored=%v)", restore)
				}
			}
			messages, err := store.ContextMessages(ctx, session, 0, 16*1024*1024, true, nil)
			if err != nil {
				t.Fatal(err)
			}
			last := messages[len(messages)-1]
			if last.Role != "assistant" || last.Content != wantAnswer || last.Incomplete != (tc.failure != "") {
				t.Fatalf("unexpected journaled reply: role=%s incomplete=%v text=%q", last.Role, last.Incomplete, last.Content)
			}
		})
	}
}

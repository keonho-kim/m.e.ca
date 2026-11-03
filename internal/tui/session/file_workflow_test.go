// Purpose: Verify active-agent file browsing, confirmed steering and queued requests preserve selected paths.
// Pattern: Local HTTP and SQLite workflow test with controlled execution barriers.
// Usage: go test -race ./internal/tui/session -run TestFileBrowserDuringAgentWork.
// Related: internal/tui/session/file_submission.go, internal/tui/session/file_selection.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"github.com/keonho-kim/meca/internal/tui/composer"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestFileBrowserDuringAgentWork(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	a.ctx = ctx
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce, selectionOnce sync.Once
	releaseTask := func() { releaseOnce.Do(func() { close(release) }) }
	blocked := &selectionBarrier{Store: a.opts.Store, started: make(chan struct{}), release: make(chan struct{})}
	a.opts.Store = blocked
	releaseSelection := func() { selectionOnce.Do(func() { close(blocked.release) }) }
	var requests atomic.Int32
	steeringText, queuedText := "review this selected file now", "summarize the selected file afterwards"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		attempt := requests.Add(1)
		if attempt > 3 {
			t.Error("unexpected extra generation")
			http.Error(w, "unexpected generation", http.StatusBadRequest)
			return
		}
		if attempt > 1 {
			last := ""
			for _, message := range request.Messages {
				if message.Role == "user" {
					last = message.Content
				}
			}
			want := steeringText
			if attempt == 3 {
				want = queuedText
			}
			if !strings.Contains(last, want) || !strings.Contains(last, "README.md") {
				t.Errorf("request %d lost input or selected file: %s", attempt, last)
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		if attempt == 1 {
			fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"working\"}}]}\n\n")
			w.(http.Flusher).Flush()
			close(started)
			select {
			case <-release:
			case <-r.Context().Done():
				return
			}
		}
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer func() {
		cancel()
		releaseTask()
		releaseSelection()
		a.files.reads.Wait()
		a.workers.Wait()
		if a.connection.client != nil {
			a.connection.client.Close()
		}
		server.Close()
	}()
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.connection.client, err = httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	a.connection.selection = connection.Selection{Connection: conn, Model: "test-model"}
	if err := os.WriteFile(filepath.Join(a.opts.Session.ProjectRoot, "README.md"), []byte("# Active project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	a.startRequest(composer.Capture("initial task", a.selectedPaths()))
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	a.compose.editor.Set("!dir")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	finishBrowserIO(t, a)
	if a.page != "files" || !a.work.state.active() {
		t.Fatal("file command was swallowed or interrupted the agent")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	finishVisibleReads(t, a)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "R", tcell.ModShift))
	a.keyEvent(tcell.NewEventKey(tcell.KeyEscape, "", 0))
	if !a.work.state.active() || a.files.filePreview != nil {
		t.Fatal("preview controls interrupted the agent")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
	select {
	case <-blocked.started:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	a.compose.editor.Set(steeringText)
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if !a.files.fileSendConfirm || !strings.Contains(modelScreen(a), "Steer this request? (y/N)") || keymapLabels(t, a)["y"] != "Steer" {
		t.Fatal("active file request did not explain steering")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.compose.fileSubmission == nil || len(a.compose.steering) != 0 {
		t.Fatal("steering bypassed pending file-selection persistence")
	}
	releaseSelection()
	finishBrowserIO(t, a)
	if a.page != "chat" || len(a.compose.steering) != 1 || !a.files.browser.Selected["README.md"] || requests.Load() != 1 {
		t.Fatal("confirmed file selection did not become steering for the active request")
	}
	a.compose.editor.Set("!dir")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	finishBrowserIO(t, a)
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	a.compose.editor.Set(queuedText)
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	if !a.files.fileSendConfirm || !a.files.fileReserve || !strings.Contains(modelScreen(a), "Queue this request? (y/N)") {
		t.Fatal("file composer did not ask to queue the request")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "N", 0))
	if a.page != "files" || a.compose.editor.String() != queuedText || len(a.compose.queued.Messages) != 0 {
		t.Fatal("cancelled file queue changed the draft")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyTab, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || len(a.compose.queued.Messages) != 1 || a.compose.queued.Messages[0].Text != queuedText || requests.Load() != 1 {
		t.Fatal("queued request ran before the active task completed")
	}
	a.compose.editor.Set("unsent draft")
	releaseTask()
	drainTask(t, a, false)
	if requests.Load() != 3 || len(a.compose.queued.Messages) != 0 || a.compose.editor.String() != "unsent draft" {
		t.Fatal("steering, queued follow-up or unsent input was lost")
	}
	history, err := a.opts.Store.ContextMessages(ctx, a.opts.Session.ID, 0, 16*1024*1024, true)
	if err != nil {
		t.Fatal(err)
	}
	var userMessages []string
	for _, message := range history {
		if message.Role == "user" {
			userMessages = append(userMessages, message.Content)
		}
	}
	if len(userMessages) != 3 || !strings.Contains(userMessages[1], steeringText) || !strings.Contains(userMessages[2], queuedText) {
		t.Fatalf("session history lost request ordering: %v", userMessages)
	}
}

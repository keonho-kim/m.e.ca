// Purpose: Verify selected paths stay visible and file-view requests return to chat before sending.
// Pattern: Virtual terminal and local HTTP contract tests.
// Usage: go test ./internal/tui/session -run TestFileComposer.
// Related: internal/tui/session/files.go, internal/tui/session/file_selection.go, internal/tui/session/render_files.go.
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
	"testing"

	"github.com/gdamore/tcell/v3"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestFileComposerShowsProjectRelativeReferences(t *testing.T) {
	a := fixture(t)
	path := filepath.Join(a.opts.Session.ProjectRoot, "src")
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "main.py"), []byte("print('hello')"), 0600); err != nil {
		t.Fatal(err)
	}
	a.openFiles("src")
	finishBrowserIO(t, a)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
	finishBrowserIO(t, a)
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	a.editor.Set("review this\nthen explain")
	for _, page := range []string{"files", "chat"} {
		a.page = page
		shown := modelScreen(a)
		for _, want := range []string{"src/main.py", "review this", "then explain"} {
			if !strings.Contains(shown, want) {
				t.Fatalf("%s missing %q: %s", page, want, shown)
			}
		}
		if strings.Contains(shown, a.opts.Session.ProjectRoot) || a.editor.String() != "review this\nthen explain" {
			t.Fatal("references changed the draft or exposed an absolute project path")
		}
	}
}

func TestFileComposerEnterSendsAfterPendingSelections(t *testing.T) {
	a := fixture(t)
	for _, path := range []string{"first", "second"} {
		if err := os.Mkdir(filepath.Join(a.opts.Session.ProjectRoot, path), 0700); err != nil {
			t.Fatal(err)
		}
	}
	requests := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
		}
		var content string
		for _, message := range request.Messages {
			if message.Role == "user" {
				content += message.Content + "\n"
			}
		}
		requests <- content
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"done"},"finish_reason":"stop"}]}`)
	}))
	t.Cleanup(server.Close)
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	a.selection = connection.Selection{Connection: conn, Model: "test-model"}
	blocked := &selectionBarrier{Store: a.opts.Store, started: make(chan struct{}), release: make(chan struct{})}
	a.opts.Store = blocked
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(blocked.release) }) }
	t.Cleanup(release)
	a.openFiles(".")
	finishBrowserIO(t, a)
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
	<-blocked.started
	a.keyEvent(tcell.NewEventKey(tcell.KeyDown, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, " ", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyBacktab, "", 0))
	a.editor.Set("review selected folders")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	if a.page != "files" || a.worker.active() || !a.fileSendConfirm {
		t.Fatal("Enter did not ask for confirmation before sending")
	}
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.worker.active() {
		t.Fatal("Enter did not return to chat and wait for selection persistence")
	}
	release()
	finishBrowserIO(t, a)
	drainTask(t, a, false)
	select {
	case request := <-requests:
		for _, want := range []string{"review selected folders", "first/", "second/"} {
			if !strings.Contains(request, want) {
				t.Fatalf("request missing %q: %s", want, request)
			}
		}
	default:
		t.Fatal("file composer did not send its request")
	}
}

func TestFileComposerQuitTextIsAMessage(t *testing.T) {
	a := fixture(t)
	a.page, a.browser.Input = "files", true
	a.worker = workerAgent
	a.steering = make(chan string, 1)
	a.editor.Set(":q")
	a.keyEvent(tcell.NewEventKey(tcell.KeyEnter, "", 0))
	a.keyEvent(tcell.NewEventKey(tcell.KeyRune, "y", 0))
	if a.page != "chat" || a.exit {
		t.Fatal("file input interpreted :q as a navigation command")
	}
	select {
	case request := <-a.steering:
		if request != ":q" {
			t.Fatal(request)
		}
	default:
		t.Fatal(":q was not sent as literal request text")
	}
}

type selectionBarrier struct {
	Store
	started, release chan struct{}
	once             sync.Once
	saveErr          error
}

func (s *selectionBarrier) SelectFile(ctx context.Context, id, path string, selected bool) error {
	s.once.Do(func() { close(s.started) })
	select {
	case <-s.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	if s.saveErr != nil {
		return s.saveErr
	}
	return s.Store.SelectFile(ctx, id, path, selected)
}

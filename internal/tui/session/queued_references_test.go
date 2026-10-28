// Purpose: Preserve selected paths at reservation time through later selection changes and editing.
// Pattern: Local HTTP workflow regression test.
// Usage: go test ./internal/tui/session -run TestQueuedRequestKeepsReferences.
// Related: internal/tui/session/pending.go, internal/tui/composer/request.go.
package session

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
)

func TestQueuedRequestKeepsReferences(t *testing.T) {
	a := fixture(t)
	requests := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct{ Messages []modelapi.Message }
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Error(err)
			return
		}
		for i := len(request.Messages) - 1; i >= 0; i-- {
			if request.Messages[i].Role == "user" {
				requests <- request.Messages[i].Content
				break
			}
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"done\"},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	}))
	defer server.Close()
	conn, err := connection.New(connection.OpenRouter, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	a.client, err = httpclient.NewClient(conn, "")
	if err != nil {
		t.Fatal(err)
	}
	defer a.client.Close()
	a.selection = connection.Selection{Connection: conn, Model: "test"}
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(a.opts.Session.ProjectRoot, name), []byte(name), 0600); err != nil {
			t.Fatal(err)
		}
	}
	a.worker = workerAgent
	a.browser.Selected = map[string]bool{"a.txt": true}
	a.reserveMessage("review the selected file")
	a.browser.Selected = map[string]bool{"b.txt": true}
	a.referenceRevision++
	a.finishWorker(notice{Done: true, RunStatus: runrecord.Answered})
	drainTask(t, a, false)
	select {
	case got := <-requests:
		if !strings.Contains(got, "a.txt") || strings.Contains(got, "b.txt") {
			t.Fatalf("queued request used a later selection: %q", got)
		}
	default:
		t.Fatal("queued request was not executed")
	}
}

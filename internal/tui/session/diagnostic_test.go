// Purpose: Verify explicit diagnostics run separately from model selection and preserve the selected model.
// Pattern: Worker lifecycle integration test with a local provider.
// Usage: go test ./internal/tui/session -run TestExplicitDiagnostic.
// Related: internal/tui/session/diagnostic.go, internal/provider/diagnostic/check.go.
package session

import (
	"fmt"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/httpclient"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestExplicitDiagnostic(t *testing.T) {
	a := fixture(t)
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"choices":[{"message":{"content":"Hello"},"finish_reason":"stop"}]}`)
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
	a.selection = connection.Selection{Connection: conn, Model: "arbitrary-alias"}
	a.diagnoseModel()
	drainTask(t, a, false)
	if posts.Load() != 1 || a.busy || a.selection.Model != "arbitrary-alias" || a.status != "모델 연결 진단을 통과했습니다" {
		t.Fatalf("posts=%d selection=%+v status=%s", posts.Load(), a.selection, a.status)
	}
}

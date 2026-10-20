// Purpose: Verify opaque serving aliases and explicit Thinking preferences transfer without response generation.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup.
// Related: internal/provider/setup/setup.go, internal/provider/diagnostic/control.go.
package setup

import (
	"context"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrepareAliasTransfersThinkingWithoutGenerating(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("setup unexpectedly generated a response: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	connection, err := modelconn.New(modelconn.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	on := true
	prepared, err := Prepare(context.Background(), connection, "", "team-coder", Defaults{Thinking: &on}, modelconn.Selection{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if prepared.Client == nil || prepared.Selection == nil {
		t.Fatal("connection ownership not transferred")
	}
	t.Cleanup(prepared.Client.Close)
	selection := prepared.Selection
	if selection.Model != "team-coder" || !selection.ThinkingSupported {
		t.Fatalf("selection=%+v", selection)
	}
	if selection.Thinking == nil || !*selection.Thinking {
		t.Fatalf("declared/default Thinking=%+v", selection)
	}
}

func TestPrepareDefaultLeavesReasoningToServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected generation: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()
	conn, err := modelconn.New(modelconn.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(context.Background(), conn, "", "models/bin/model114", Defaults{}, modelconn.Selection{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Client.Close()
	if prepared.Selection.Thinking != nil || prepared.Selection.Effort() != "" || !prepared.Selection.ThinkingSupported {
		t.Fatalf("selection=%+v", prepared.Selection)
	}
}

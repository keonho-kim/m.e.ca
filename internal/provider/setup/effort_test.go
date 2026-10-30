// Purpose: Verify server defaults and explicit session preferences are revalidated during connection setup.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup -run TestEffort.
// Related: internal/provider/setup/setup.go, internal/provider/connection/reasoning_options.go.
package setup

import (
	"context"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEffortSetupDefaultsAndSessionOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if rejectImageProbe(t, w, r) {
			return
		}
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
	defaults := Defaults{}
	for _, choice := range []*string{nil, ptr("low"), ptr("")} {
		off := false
		previous := modelconn.Selection{Connection: connection, Model: "team-coder", Thinking: &off, ReasoningEffort: choice, Details: &modelconn.ModelInfo{ID: "team-coder"}}
		prepared, err := Prepare(context.Background(), connection, "", "team-coder", defaults, previous, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(prepared.Client.Close)
		want := ""
		if choice != nil {
			want = *choice
		}
		if got := prepared.Selection.Effort(); got != want {
			t.Fatalf("effort=%q want %q", got, want)
		}
		if prepared.Selection.Thinking == nil || *prepared.Selection.Thinking {
			t.Fatal("effort implicitly enabled Thinking")
		}
	}
}

func ptr(value string) *string { return &value }

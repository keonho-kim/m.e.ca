// Purpose: Verify effort defaults and explicit session preferences are revalidated during connection setup.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup -run TestEffort.
// Related: internal/provider/setup/setup.go, internal/provider/profile/effort.go.
package setup

import (
	"context"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEffortSetupDefaultsAndSessionOverride(t *testing.T) {
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
	defaults := Defaults{Profile: "qwen3.8-27b", ReasoningEffort: "medium"}
	for _, choice := range []*string{nil, ptr("low"), ptr("")} {
		previous := modelconn.Selection{Connection: connection, Model: "team-coder", Profile: defaults.Profile, ReasoningEffort: choice}
		prepared, err := Prepare(context.Background(), connection, "", "team-coder", defaults, previous, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(prepared.Client.Close)
		want := "medium"
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

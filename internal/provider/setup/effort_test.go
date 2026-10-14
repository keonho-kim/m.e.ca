// Purpose: Verify effort defaults and explicit session preferences are revalidated during connection setup.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup -run TestEffort.
// Related: internal/provider/setup/setup.go, internal/provider/diagnostic/effort.go.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEffortSetupDefaultsAndSessionOverride(t *testing.T) {
	efforts := make(chan string, 16)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct {
			Effort   string         `json:"reasoning_effort"`
			Template map[string]any `json:"chat_template_kwargs"`
			Messages []any          `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		reply, reason := "TOOL read_file", ""
		if len(body.Messages) > 2 {
			reply = "AGENTS.md"
		}
		if body.Template["enable_thinking"] == true {
			reply = "391"
			reason = "17 times 23 is 391"
		}
		if body.Effort != "" {
			efforts <- body.Effort
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply, "reasoning": reason}, "finish_reason": "stop"}}}); err != nil {
			t.Error(err)
		}
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
	var observed []string
	for len(efforts) > 0 {
		observed = append(observed, <-efforts)
	}
	if fmt.Sprint(observed) != "[medium low]" {
		t.Fatalf("effort probes=%v", observed)
	}
}

func ptr(value string) *string { return &value }

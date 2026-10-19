// Purpose: Verify aliased profiles and declared Thinking controls transfer into selected clients.
// Pattern: Provider setup integration tests.
// Usage: go test ./internal/provider/setup.
// Related: internal/provider/setup/setup.go, internal/provider/diagnostic/check.go.
package setup

import (
	"context"
	"encoding/json"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPrepareAliasTransfersVerifiedProfileAndThinking(t *testing.T) {
	for _, reasoningAvailable := range []bool{false, true} {
		t.Run(map[bool]string{false: "unverified", true: "verified"}[reasoningAvailable], func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				var body struct {
					Model    string `json:"model"`
					Messages []struct {
						Content string `json:"content"`
					} `json:"messages"`
					Template map[string]bool `json:"chat_template_kwargs"`
				}
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
					return
				}
				if body.Model != "team-coder" {
					t.Errorf("serving alias rewritten: %s", body.Model)
				}
				on, declared := body.Template["enable_thinking"]
				if !declared {
					t.Error("profile switch missing")
				}
				reply, reason := "Hello!", ""
				if on {
					reply = "391"
					if reasoningAvailable {
						reason = "17 times 23 is 391"
					}
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
			prepared, err := Prepare(context.Background(), connection, "", "team-coder", Defaults{Profile: "ornith-1.5-9b", Thinking: true}, modelconn.Selection{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			if prepared.Client == nil || prepared.Selection == nil {
				t.Fatal("connection ownership not transferred")
			}
			t.Cleanup(prepared.Client.Close)
			selection := prepared.Selection
			if selection.Profile != "ornith-1.5-9b" || selection.Model != "team-coder" || !selection.ThinkingSupported {
				t.Fatalf("selection=%+v", selection)
			}
			if selection.Thinking == nil || !*selection.Thinking {
				t.Fatalf("unverified/default Thinking=%+v", selection)
			}
		})
	}
}

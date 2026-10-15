// Purpose: Verify metadata-driven profiles keep aliases and reject conflicting or routed identities.
// Pattern: Setup behavior tests.
// Usage: go test ./internal/provider/setup -run TestIdentity.
// Related: internal/provider/setup/identity.go, internal/provider/setup/setup.go.
package setup

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/identity"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIdentityAutomaticProfileKeepsWireAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/models" {
			fmt.Fprint(w, `{"data":[{"id":"models/bin/modelX","root":"Qwen/Qwen3.8-27B"}]}`)
			return
		}
		var body struct {
			Model    string
			Messages []any
			Template map[string]bool `json:"chat_template_kwargs"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		if body.Model != "models/bin/modelX" {
			t.Errorf("wire alias=%s", body.Model)
		}
		on, present := body.Template["enable_thinking"]
		if !present {
			t.Error("identified profile not applied")
		}
		reply, reasoning := "Hello!", ""
		if on {
			reply, reasoning = "391", "calculated"
		}
		if err := json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": reply, "reasoning": reasoning}, "finish_reason": "stop"}}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	conn, err := connection.New(connection.VLLM, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	prepared, err := Prepare(context.Background(), conn, "", "models/bin/modelX", Defaults{}, connection.Selection{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer prepared.Client.Close()
	if prepared.Selection.Profile != "qwen3.8-27b" || prepared.Selection.Identity == nil || !prepared.Selection.ThinkingSupported {
		t.Fatalf("selection=%+v", prepared.Selection)
	}
}

func TestIdentityResolutionSafety(t *testing.T) {
	for _, tc := range []struct {
		name, original, override, want string
		routed, wantError              bool
	}{
		{"canonical", "Qwen/Qwen3.8-27B", "", "qwen3.8-27b", false, false},
		{"repository URL", "https://huggingface.co/Qwen/Qwen3.8-27B", "", "qwen3.8-27b", false, false},
		{"repository subpath", "https://huggingface.co/Qwen/Qwen3.8-27B/tree/main", "", "generic", false, false},
		{"cache", "/cache/models--Qwen--Qwen3.8-27B/snapshots/abc", "", "qwen3.8-27b", false, false},
		{"local filename", "/models/Qwen3.8-27B.gguf", "", "generic", false, false},
		{"unknown", "/models/bin/modelX", "", "generic", false, false},
		{"manual conflict", "Qwen/Qwen3.8-27B", "ornith-1.5-9b", "", false, true},
		{"routed automatic", "Qwen/Qwen3.8-27B", "", "generic", true, false},
		{"routed manual", "Qwen/Qwen3.8-27B", "qwen3.8-27b", "", true, true},
		{"disabled", "Qwen/Qwen3.8-27B", "generic", "generic", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := identity.Report{Routed: tc.routed}
			report.Add("server", "original", tc.original)
			p, err := resolveIdentity("alias", tc.override, &report)
			if (err != nil) != tc.wantError || err == nil && p.ID != tc.want {
				t.Fatalf("profile=%s err=%v", p.ID, err)
			}
		})
	}
}

func TestIdentityFlashNextAliasAndFP8Agree(t *testing.T) {
	report := identity.Report{}
	report.Add("server", "original", "Qwen/Qwen3.8-Flash-Next-FP8")
	selected, err := resolveIdentity("Qwen-3.8-180B", "", &report)
	if err != nil || selected.ID != "qwen3.8-flash-next" || report.ProfileSource != "server" {
		t.Fatalf("%+v %v", selected, err)
	}
}

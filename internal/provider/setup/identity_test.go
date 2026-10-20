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

func TestIdentityIncompleteMetadataKeepsConflictChecks(t *testing.T) {
	for _, tc := range []struct {
		name, served, original, override, want string
		wantError                              bool
	}{
		{"automatic fallback", "Qwen/Qwen3.8-27B", "", "", "generic", false},
		{"manual without original", "alias", "", "qwen3.8-27b", "qwen3.8-27b", false},
		{"matching original", "alias", "Qwen/Qwen3.8-27B", "qwen3.8-27b", "qwen3.8-27b", false},
		{"conflicting original", "alias", "Qwen/Qwen3.8-27B", "ornith-1.5-9b", "", true},
		{"conflicting served ID", "Qwen/Qwen3.8-27B", "", "ornith-1.5-9b", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := identity.Report{Incomplete: true}
			report.Add("server", "original", tc.original)
			selected, err := resolveIdentity(tc.served, tc.override, &report)
			if (err != nil) != tc.wantError || err == nil && selected.ID != tc.want {
				t.Fatalf("profile=%s err=%v", selected.ID, err)
			}
			if err == nil && tc.override != "" && report.ProfileSource != "manual" {
				t.Fatalf("profile source=%s", report.ProfileSource)
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

func TestIdentityKnownServedNameWithUnrecognizedRoot(t *testing.T) {
	for _, tc := range []struct {
		name, served, original, want string
		wantError                    bool
	}{
		{"Ornith local root", "ornith-1.5-35B-A3B", "/models/ornith-local", "ornith-1.5-35b-a3b", false},
		{"Qwen local root", "Qwen/Qwen3.8-27B", "/models/modelX", "qwen3.8-27b", false},
		{"opaque alias", "modelX", "/models/Ornith-1.5-35B-A3B", "generic", false},
		{"recognized conflict", "Ornith-1.5-35B-A3B", "Qwen/Qwen3.8-27B", "", true},
		{"excluded checkpoint conflict", "Ornith-1.5-35B-A3B", "ornith-ai/Ornith-1.5-397B", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := identity.Report{}
			report.Add("/v1/models.root", "original", tc.original)
			selected, err := resolveIdentity(tc.served, "", &report)
			if (err != nil) != tc.wantError || err == nil && selected.ID != tc.want {
				t.Fatalf("profile=%s err=%v", selected.ID, err)
			}
			if err == nil && tc.want != "generic" && report.ProfileSource != "catalogue" {
				t.Fatalf("incorrect identity basis: %s", report.ProfileSource)
			}
		})
	}
}

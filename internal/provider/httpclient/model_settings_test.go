// Purpose: Verify native reasoning metadata, literal effort values and model output limits.
// Pattern: HTTP contract tests.
// Usage: go test ./internal/provider/httpclient -run TestNativeModelSettings.
// Related: internal/provider/discovery/model_settings.go, internal/provider/protocol/controls.go.
package httpclient

import (
	"context"
	"encoding/json"
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/protocol"
	"net/http"
	"testing"
)

func TestNativeModelSettings(t *testing.T) {
	calls := 0
	c := mockClient(t, connection.Ollama, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/ps" {
			fmt.Fprint(w, `{"models":[]}`)
			return
		}
		if r.URL.Path == "/api/show" {
			fmt.Fprint(w, `{"thinking":{"values":[false,"Light","Deep"],"default":"Light"}}`)
			return
		}
		calls++
		var body struct {
			Think   string
			Options struct {
				NumPredict int `json:"num_predict"`
			}
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body.Think != "Deep" {
			t.Errorf("literal effort lost: %q", body.Think)
		}
		fmt.Fprintln(w, `{"message":{"thinking":"reason","content":"answer"},"done":true}`)
	})
	info, err := prepareModel(c, context.Background(), "alias")
	if err != nil {
		t.Fatal(err)
	}
	if info.Reasoning == nil || len(info.Reasoning.Levels) != 2 || !info.Reasoning.Off {
		t.Fatalf("metadata=%+v", info)
	}
	on := true
	request := modelapi.Request{ModelID: "alias", MaxOutput: 100, Thinking: &on, ReasoningEffort: "Deep", Messages: []modelapi.Message{{Role: "user", Content: "hi"}}}
	if err := c.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err != nil {
		t.Fatal(err)
	}
	request.ReasoningEffort = "deep"
	if err := c.Generate(context.Background(), request, func(modelapi.Event) error { return nil }); err == nil {
		t.Fatal("unsupported effort accepted")
	}
	if calls != 1 {
		t.Fatalf("invalid setting reached server: %d", calls)
	}
}

func TestNativeReasoningDefaultUsesAdvertisedLevel(t *testing.T) {
	c := &Client{connection: connection.Connection{Kind: connection.LMStudio}, modelInfo: &connection.ModelInfo{ID: "alias", Reasoning: &connection.Reasoning{On: true, Off: true, Levels: []string{"low", "high"}, Default: "low"}}}
	on := true
	controls, err := protocol.ControlsFor(c.connection.Kind, c.profileID, c.modelInfo, modelapi.Request{ModelID: "alias", Thinking: &on})
	if err != nil || controls.ReasoningEffort != "low" {
		t.Fatalf("default=%+v err=%v", controls, err)
	}
}

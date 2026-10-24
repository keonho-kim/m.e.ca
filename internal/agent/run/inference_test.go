// Purpose: Verify all main-agent calls use the configured generation parameters.
// Pattern: Runtime behavior tests with captured generation requests.
// Usage: go test ./internal/agent/run -run TestInference.
// Related: internal/agent/run/context.go, internal/agent/inference/settings.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"testing"
)

func checkInference(t *testing.T, request modelapi.Request, parameters inference.Parameters) {
	t.Helper()
	if request.Temperature == nil || *request.Temperature != parameters.Temperature || request.MaxOutput != parameters.MaxOutputTokens {
		t.Fatalf("request sampling=%v/%d; want %+v", request.Temperature, request.MaxOutput, parameters)
	}
}
func TestInferenceMainParametersStayStable(t *testing.T) {
	g := &generator{replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "verify"}), "Verified"}}
	r := fixture(g, &executor{}, &journal{})
	settings := inference.Defaults()
	settings.Agent.Temperature = 0
	r.Inference = &settings
	r.ProjectInstructions = "Project rule: do not remove tests."
	templates := prompt.Defaults()
	templates.System += "\nMAIN-AGENT-ONLY"
	r.Prompts = &templates
	result, err := r.Run(context.Background(), "change and verify", approve, emit)
	if err != nil || result.Status != Answered {
		t.Fatalf("%+v %v", result, err)
	}
	for _, request := range g.requests {
		checkInference(t, request, settings.Agent)
	}
	if len(g.requests) != 3 {
		t.Fatalf("unexpected model calls: %d", len(g.requests))
	}
}

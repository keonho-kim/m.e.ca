// Purpose: Verify execution phases select role settings without inheriting the previous call's sampling.
// Pattern: Runtime behavior tests with captured generation requests.
// Usage: go test ./internal/agent/run -run TestInference.
// Related: internal/agent/run/context.go, internal/agent/inference/settings.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"strings"
	"testing"
)

func checkInference(t *testing.T, request modelapi.Request, parameters inference.Parameters) {
	t.Helper()
	if request.Temperature == nil || *request.Temperature != parameters.Temperature || request.MaxOutput != parameters.MaxOutputTokens {
		t.Fatalf("request sampling=%v/%d; want %+v", request.Temperature, request.MaxOutput, parameters)
	}
}
func TestInferenceToolPhasesAndConclave(t *testing.T) {
	g := &generator{purposes: []string{"check"}, replies: []string{nativeCall("write_file", map[string]any{"path": "a", "content": "new"}), nativeCall("execute_command", map[string]any{"command": "verify"}), "Verified"}}
	r := fixture(g, &executor{}, &journal{})
	settings := inference.Defaults()
	settings.Agent.Temperature = 0
	settings.Conclave.MaxOutputTokens = 1234
	r.Inference = &settings
	r.ProjectInstructions = "Project rule: do not remove tests."
	templates := prompt.Defaults()
	templates.System += "\nMAIN-AGENT-ONLY"
	templates.Conclave += "\nCUSTOM-REVIEW-GUIDANCE"
	r.Prompts = &templates
	result, err := r.Run(context.Background(), "change and verify", false, approve, emit)
	if err != nil || result.Status != Verified {
		t.Fatalf("%+v %v", result, err)
	}
	for i, role := range []inference.Role{inference.Agent, inference.Agent, inference.Agent} {
		p, err := inference.Resolve(&settings, role)
		if err != nil {
			t.Fatal(err)
		}
		checkInference(t, g.requests[i], p)
	}
	if len(g.reviewRequests) != 1 {
		t.Fatalf("reviews=%d", len(g.reviewRequests))
	}
	for _, request := range g.reviewRequests {
		checkInference(t, request, settings.Conclave)
		system := request.Messages[0].Content
		if strings.Contains(system, "MAIN-AGENT-ONLY") || !strings.Contains(system, "CUSTOM-REVIEW-GUIDANCE") || !strings.Contains(system, "do not remove tests") {
			t.Fatalf("mixed review prompt: %s", system)
		}
	}
}

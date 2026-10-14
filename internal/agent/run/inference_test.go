// Purpose: Verify task phases select role settings without inheriting the previous call's sampling.
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
	g := &generator{replies: []string{"TOOL write_file", "a", "new", "CHECK execute_command", "verify", "DONE\nVerified"}}
	r := fixture(g, &executor{}, &journal{})
	settings := inference.Defaults()
	settings.PFCValue.Temperature = 0
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
	for i, role := range []inference.Role{inference.Agent, inference.PFCValue, inference.PFCContent, inference.Agent, inference.PFCContent, inference.Agent} {
		p, err := inference.Resolve(&settings, role)
		if err != nil {
			t.Fatal(err)
		}
		checkInference(t, g.requests[i], p)
	}
	for _, request := range g.groundingRequests {
		checkInference(t, request, settings.Grounding)
	}
	if len(g.reviewRequests) != 3 {
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
func TestInferenceGoalAndReport(t *testing.T) {
	g := &generator{replies: []string{"GOAL: explain\nTASK answer: first\nTASK answer: second", "DONE\none", "DONE\ntwo", "DONE\nfinal"}}
	r := fixture(g, &executor{}, &journal{})
	settings := inference.Defaults()
	settings.Goal.Temperature = 0.3
	settings.Report.Temperature = 0.1
	r.Inference = &settings
	if _, err := r.RunGoal(context.Background(), "explain", approve, emit); err != nil {
		t.Fatal(err)
	}
	checkInference(t, g.requests[0], settings.Goal)
	checkInference(t, g.requests[3], settings.Report)
	g = &generator{replies: []string{"DONE\nplan"}}
	r = fixture(g, &executor{}, &journal{})
	r.Inference = &settings
	if _, err := r.Run(context.Background(), "plan", true, approve, emit); err != nil {
		t.Fatal(err)
	}
	checkInference(t, g.requests[0], settings.Plan)
}

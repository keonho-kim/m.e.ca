// Purpose: Preserve role-specific output budgets while fitting independent evidence.
// Pattern: Scripted runtime boundary test.
// Usage: go test ./internal/agent/run -run TestLATOutputBudgets.
// Related: internal/agent/run/lat_fork.go, internal/agent/lat/packet.go.
package run

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/inference"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

type latBudgetModel struct {
	*latModel
	summaries atomic.Int32
}

func (m *latBudgetModel) Generate(_ context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
	if len(request.Messages) < 2 || !strings.HasPrefix(request.Messages[0].Content, "# Preserve requirements and observed work for review") || !strings.Contains(request.Messages[1].Content, "SOURCE_EVIDENCE_END") {
		return fmt.Errorf("summary did not receive the independent original")
	}
	m.summaries.Add(1)
	return send(modelapi.Event{Delta: "SUMMARY_EVIDENCE_PRESENT", Done: true})
}

func TestLATOutputBudgetsPreserveEvidenceAndRoleSettings(t *testing.T) {
	const window = 262144
	for _, tc := range []struct {
		name        string
		agent, lat  int
		evidenceLen int
	}{
		{name: "automatic", agent: 0, lat: 0, evidenceLen: 35000},
		{name: "explicit", agent: 180000, lat: 1 << 20, evidenceLen: 50000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, scripted, _, store := latFixture("Completed answer")
			model := &latBudgetModel{latModel: scripted}
			r.Generator, r.ContextWindow = model, window
			settings := inference.Defaults()
			settings.Agent.MaxOutputTokens, settings.Agent.Temperature = tc.agent, 0.71
			settings.LAT.MaxOutputTokens, settings.LAT.Temperature = tc.lat, 0.13
			r.Inference = &settings
			store.messages = []modelapi.Message{{Role: "user", Content: strings.Repeat("e", tc.evidenceLen) + "SOURCE_EVIDENCE_END"}}
			result, err := r.Run(context.Background(), "Implement the request", approve, emit)
			if err != nil || result.Text != "Completed answer" || model.summaries.Load() != 1 {
				t.Fatalf("output reservation did not preserve oversized evidence: result=%+v summaries=%d err=%v", result, model.summaries.Load(), err)
			}
			reserve, err := modelapi.OutputBudget(tc.agent, window, 0, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, phase := range []string{"plan", "critique", "refine", "review"} {
				requests := scripted.requests[phase]
				if len(requests) != 1 {
					t.Fatalf("%s generated %d requests", phase, len(requests))
				}
				request := requests[0]
				assertContains(t, request, "SUMMARY_EVIDENCE_PRESENT")
				output, temperature := reserve, 0.71
				if phase == "plan" || phase == "refine" {
					if request.MaxOutput != tc.agent {
						t.Errorf("%s changed agent output setting: %d", phase, request.MaxOutput)
					}
				} else {
					output, temperature = request.MaxOutput, 0.13
					if output < 1 || output > window/4 || tc.lat == 0 && output != 4096 {
						t.Errorf("%s has invalid LAT output cap: %d", phase, output)
					}
				}
				if request.Temperature == nil || *request.Temperature != temperature {
					t.Errorf("%s used the wrong sampling role", phase)
				}
				if size := modelapi.TextBytes(request) + int64(output); size > window {
					t.Errorf("%s evidence and output exceed context: %d > %d", phase, size, window)
				}
			}
		})
	}
}

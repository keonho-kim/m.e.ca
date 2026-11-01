// Purpose: Preserve current requirements and isolated evidence within final review budgets.
// Pattern: Context boundary regression tests.
// Usage: go test ./internal/agent/lat.
// Related: internal/agent/lat/packet.go, internal/agent/lat/memory/tail.go.
package lat

import (
	"encoding/json"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	"strings"
	"testing"
)

func TestPacketPreservesIndependentEvidence(t *testing.T) {
	input := Input{Prompt: prompt.Defaults().LAT, Instructions: "PROJECT_RULE", Objective: "이제 구현해줘", Context: json.RawMessage(`{"independent_review_memory":"Create a simple API","later_original_records_oldest_first":[{"kind":"outcome","text":"actual result"}]}`), Draft: strings.Repeat("작업 결과 ", 10000), ContextWindow: 8192, MaxOutput: 4096}
	packet, err := buildPacket(input)
	if err != nil {
		t.Fatal(err)
	}
	if packet.MaxOutput != 2048 || !json.Valid([]byte(packet.Content)) {
		t.Fatal("review packet or budget changed")
	}
	system, err := prompt.Compose(input.Prompt, input.Instructions)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"PROJECT_RULE", "이제 구현해줘", "Create a simple API", "actual result"} {
		if !strings.Contains(system+packet.Content, value) {
			t.Fatal("missing requirement/evidence", value)
		}
	}
	size := packetSize(t, input, packet)
	if size > 8192 {
		t.Fatal("exceeded context", size)
	}
}

func TestPacketNeverClipsCurrentUserRequirement(t *testing.T) {
	objective := strings.Repeat("requirement ", 1400) + "MANDATORY_END"
	packet, err := buildPacket(Input{Prompt: prompt.Defaults().LAT, Objective: objective, Context: json.RawMessage(`{}`), Draft: "done", ContextWindow: 65536})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(packet.Content, "MANDATORY_END") {
		t.Fatal("current user requirement truncated")
	}
	_, err = buildPacket(Input{Prompt: prompt.Defaults().LAT, Objective: objective, Context: json.RawMessage(`{}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("unfittable mandatory request silently clipped")
	}
}

func TestPacketRejectsUnfittableIndependentEvidence(t *testing.T) {
	_, err := buildPacket(Input{Prompt: prompt.Defaults().LAT, Objective: "work", Context: json.RawMessage(`{"text":"` + strings.Repeat("x", 100000) + `"}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("oversized evidence silently omitted")
	}
	_, err = buildPacket(Input{Prompt: prompt.Defaults().LAT, Instructions: strings.Repeat("rule", 4096), Context: json.RawMessage(`{}`), ContextWindow: 8192})
	if err == nil {
		t.Fatal("unfittable instructions accepted")
	}
}

func TestPacketPreservesMandatoryPlanAndCritique(t *testing.T) {
	input := Input{Prompt: "Review the supplied packet.", Objective: "Implement the accepted plan.", Plan: strings.Repeat("계획: \"retain every requirement\"\n", 75) + "PLAN_END", Critique: strings.Repeat("Critique: revise missing checks.\n", 75) + "CRITIQUE_END", Context: json.RawMessage(`{"evidence":"original"}`), Draft: strings.Repeat("draft\n", 20000), ContextWindow: 16384}
	built, err := buildPacket(input)
	if err != nil {
		t.Fatal(err)
	}
	var packet reviewPacket
	if err := json.Unmarshal([]byte(built.Content), &packet); err != nil {
		t.Fatal(err)
	}
	if packet.Plan != input.Plan || packet.Critique != input.Critique {
		t.Fatal("mandatory planning artifacts clipped or omitted")
	}
	if packet.Draft == input.Draft {
		t.Fatal("oversized optional draft was not excerpted")
	}
	sourceBudget, err := inputSourceBudget(input)
	if err != nil || sourceBudget < len(input.Context) {
		t.Fatalf("source budget: %d %v", sourceBudget, err)
	}
	size := packetSize(t, input, built)
	if size > input.ContextWindow {
		t.Fatalf("mandatory artifacts exceeded window: %d", size)
	}
}

func TestPacketRejectsUnfittablePlanningArtifacts(t *testing.T) {
	base := Input{Prompt: "Review the supplied packet.", Objective: "Implement.", Context: json.RawMessage(`{}`), ContextWindow: 8192}
	for _, test := range []struct{ plan, critique string }{
		{strings.Repeat("p", 6000) + "PLAN_END", ""},
		{"", strings.Repeat("c", 6000) + "CRITIQUE_END"},
		{strings.Repeat("p", 3000), strings.Repeat("c", 3000)},
	} {
		input := base
		input.Plan, input.Critique = test.plan, test.critique
		if _, err := inputSourceBudget(input); err == nil {
			t.Fatal("unfittable mandatory artifacts given a source budget")
		}
		if _, err := buildPacket(input); err == nil {
			t.Fatal("unfittable mandatory artifacts silently clipped")
		}
	}
}

func TestPacketReservesAgentOutputWithoutReviewCap(t *testing.T) {
	input := Input{Prompt: "Plan the implementation.", Objective: "Implement the request.", Context: json.RawMessage(`{}`), ContextWindow: 32768, OutputReserve: 15000}
	budget, err := inputSourceBudget(input)
	if err != nil {
		t.Fatal(err)
	}
	input.Context = json.RawMessage(`{"evidence":"` + strings.Repeat("x", budget-64) + `EVIDENCE_END"}`)
	packet, err := buildPacket(input)
	if err != nil {
		t.Fatal(err)
	}
	if packet.MaxOutput != input.OutputReserve || packet.MaxOutput <= input.ContextWindow/4 {
		t.Fatalf("agent output reserve changed: %d", packet.MaxOutput)
	}
	if budget+input.OutputReserve > input.ContextWindow || packetSize(t, input, packet) > input.ContextWindow {
		t.Fatal("evidence invaded generation reserve")
	}
	var body reviewPacket
	if err := json.Unmarshal([]byte(packet.Content), &body); err != nil {
		t.Fatal(err)
	}
	if string(body.Context) != string(input.Context) || !strings.Contains(packet.Content, "EVIDENCE_END") {
		t.Fatal("reserved output caused evidence to be clipped")
	}
	input.OutputReserve = 32768
	if _, err := buildPacket(input); err == nil {
		t.Fatal("unfittable agent output reserve accepted")
	}
}

func packetSize(t *testing.T, input Input, packet Packet) int {
	t.Helper()
	system, err := prompt.Compose(input.Prompt, input.Instructions)
	if err != nil {
		t.Fatal(err)
	}
	return 1024 + packet.MaxOutput + len(system) + len(packet.Content)
}

// These adapters keep fixture evidence separate from immutable frame preparation.
func buildPacket(input Input) (Packet, error) {
	frame, err := Prepare(input)
	if err != nil {
		return Packet{}, err
	}
	return frame.Build(input.Context)
}
func inputSourceBudget(input Input) (int, error) {
	frame, err := Prepare(input)
	if err != nil {
		return 0, err
	}
	return frame.SourceBudget(), nil
}

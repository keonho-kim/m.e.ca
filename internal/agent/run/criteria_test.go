// Purpose: Verify completion criteria persist and reach later contexts without reaching executable tools.
// Pattern: Runtime contract tests.
// Usage: go test ./internal/agent/run -run TestCriteria.
// Related: internal/agent/run/criteria.go, internal/agent/run/context.go.
package run

import (
	"context"
	"encoding/json"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	"strings"
	"testing"
)

func TestCriteriaPersistAndAreInjected(t *testing.T) {
	criteria := []string{"README describes installation", "Existing files stay unchanged"}
	g := &generator{replies: []string{
		nativeCall("read_file", map[string]any{"path": "README.md", "completion_criteria": criteria}),
		nativeCall("write_file", map[string]any{"path": "README.md", "content": "Install instructions"}),
		"Completed README; no execution tests were run",
	}}
	e, j := &executor{}, &journal{}
	result, err := fixture(g, e, j).Run(context.Background(), "create installation documentation", false, approve, emit)
	if err != nil || result.Status != Answered || e.executed != 2 {
		t.Fatalf("%+v executions=%d err=%v", result, e.executed, err)
	}
	for _, request := range g.requests[1:] {
		facts := request.Messages[len(request.Messages)-1].Content
		for _, criterion := range criteria {
			if !strings.Contains(facts, criterion) {
				t.Fatalf("criteria missing from current state: %s", facts)
			}
		}
	}
	for _, call := range e.calls {
		if strings.Contains(string(call.Arguments), "completion_criteria") {
			t.Fatal("metadata reached executable tool")
		}
	}
	persisted := false
	for _, event := range j.events {
		if event.Checkpoint == nil {
			continue
		}
		body, _ := json.Marshal(event.Checkpoint)
		if strings.Contains(string(body), "README describes installation") {
			persisted = true
		}
	}
	if !persisted {
		t.Fatal("criteria were not checkpointed")
	}
}

func TestCriteriaSurviveCompactedContext(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s := execution{runner: r, limits: DefaultLimits(), system: "rules", memoryContext: "Short summary without acceptance conditions", checkpoint: Checkpoint{Objective: "current request", Criteria: []string{"Cancellation preserves the partial answer"}}}
	request, err := s.request("next action", inference.Agent)
	if err != nil {
		t.Fatal(err)
	}
	last := request.Messages[len(request.Messages)-1].Content
	if !strings.Contains(last, "completion_criteria_model_claims") || !strings.Contains(last, "Cancellation preserves the partial answer") {
		t.Fatalf("criteria lost after history compression: %s", last)
	}
}

func TestCriteriaUpdatesDoNotMutateEarlierCheckpoints(t *testing.T) {
	g := &generator{replies: []string{
		nativeCall("read_file", map[string]any{"path": "a", "completion_criteria": []string{"Preserve initial behavior"}}),
		nativeCall("read_file", map[string]any{"path": "b", "completion_criteria": []string{"Preserve behavior including the discovered edge case"}}),
		"Inspection complete",
	}}
	j := &journal{}
	if _, err := fixture(g, &executor{}, j).Run(context.Background(), "inspect behavior", false, approve, emit); err != nil {
		t.Fatal(err)
	}
	initial, updated := false, false
	for _, event := range j.events {
		if event.Checkpoint == nil || len(event.Checkpoint.Criteria) == 0 {
			continue
		}
		switch event.Checkpoint.Criteria[0] {
		case "Preserve initial behavior":
			initial = true
		case "Preserve behavior including the discovered edge case":
			updated = true
		}
	}
	if !initial || !updated {
		t.Fatal("earlier criterion snapshot was overwritten")
	}
}

func TestCriteriaRejectMalformedMetadataBeforeExecution(t *testing.T) {
	for _, value := range []any{[]string{}, []string{"same", "same"}, []string{strings.Repeat("x", 513)}, []string{"a", "b", "c", "d"}, "not an array"} {
		g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a", "completion_criteria": value}), "Need to correct the criteria metadata"}}
		e := &executor{}
		result, err := fixture(g, e, &journal{}).Run(context.Background(), "inspect", false, approve, emit)
		if err != nil || e.prepared != 0 || result.Status != Answered {
			t.Fatalf("invalid metadata reached execution: %+v %v", result, err)
		}
	}
}

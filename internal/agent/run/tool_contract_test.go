// Purpose: Keep native tool schemas limited to execution arguments and per-call grounding.
// Pattern: Runtime contract regression tests.
// Usage: go test ./internal/agent/run -run TestToolSchemas.
// Related: internal/agent/run/context.go, internal/agent/tool/intent.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/inference"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestToolSchemasExcludeRequestControlMetadata(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	r, _, _, _, _ := realBatchRunner(t, nil)
	r.Generator = g
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	for _, definition := range g.requests[0].Tools {
		reason := false
		for _, p := range definition.Parameters {
			switch p.Name {
			case "access", "completion_criteria", "purpose", "permission", "approved":
				t.Errorf("%s exposes request control %s", definition.Name, p.Name)
			case "reason":
				reason = p.Required
			}
		}
		if !reason {
			t.Errorf("%s lost required self-grounding", definition.Name)
		}
	}
}

func TestLatestSuccessfulChangeSurvivesSubsequentRead(t *testing.T) {
	g := &generator{replies: []string{
		nativeCall("write_file", map[string]any{"path": "empty", "content": ""}),
		nativeCall("read_file", map[string]any{"path": "empty"}),
		"Created the requested empty file",
	}}
	e := &executor{outcomes: []toolapi.Result{
		{Summary: "Created file /project/empty (0 bytes)", Content: []byte(`{"changed":true,"size_bytes":0}`)},
		{Summary: "Read empty file", Content: []byte(`{"Content":"","SizeBytes":0}`)},
	}}
	if _, err := fixture(g, e, &journal{}).Run(context.Background(), "create an empty file", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[len(g.requests)-1]
	text := request.Messages[len(request.Messages)-1].Content
	if !strings.Contains(text, "last_successful_file_change") || !strings.Contains(text, "Created file /project/empty (0 bytes)") {
		t.Fatalf("latest successful change was lost: %s", text)
	}
}

func TestToolMetadataCannotChangeRequestState(t *testing.T) {
	for _, name := range []string{"access", "completion_criteria", "permission", "approved", "purpose"} {
		t.Run(name, func(t *testing.T) {
			g := &generator{replies: []string{nativeCall("read_file", map[string]any{"path": "a", name: "unexpected"}), "Invalid arguments; no operation executed."}}
			e := &executor{}
			result, err := fixture(g, e, &journal{}).Run(context.Background(), "inspect a", approve, emit)
			if err != nil || e.prepared != 0 || result.Status != Answered {
				t.Fatalf("unexpected metadata reached execution: %+v %v", result, err)
			}
		})
	}
}
func TestToolContextKeepsOriginalRequestAfterCompaction(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s := execution{runner: r, limits: DefaultLimits(), system: "rules", memoryContext: "Old request summary", checkpoint: Checkpoint{Objective: "Change only the second line; preserve the rest."}}
	request, err := s.request("next action", inference.Agent)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range request.Messages {
		found = found || strings.Contains(m.Content, s.checkpoint.Objective)
	}
	if !found {
		t.Fatal("original request lost after compaction")
	}
}

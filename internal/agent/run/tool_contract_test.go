// Purpose: Keep native tool schemas limited to execution arguments and per-call grounding.
// Pattern: Runtime contract regression tests.
// Usage: go test ./internal/agent/run -run TestToolSchemas.
// Related: internal/agent/run/context.go, internal/agent/tool/intent.go.
package run

import (
	"context"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"strings"
	"testing"
)

func TestToolSchemasRequireGrounding(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	r, _, _, _, _ := realBatchRunner(t, nil)
	r.Generator = g
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	for _, definition := range g.requests[0].Tools {
		reason := false
		for _, p := range definition.Parameters {
			if p.Name == "reason" {
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
		nativeCall("inspect_fixture", map[string]any{"path": "empty"}),
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
	if !strings.Contains(text, "last_change") || !strings.Contains(text, "Created file /project/empty (0 bytes)") {
		t.Fatalf("latest successful change was lost: %s", text)
	}
}

func TestToolContextKeepsOriginalRequestAfterCompaction(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s := execution{runner: r, limits: DefaultLimits(), system: "rules", memoryContext: "Old request summary", checkpoint: Checkpoint{Objective: "Change only the second line; preserve the rest."}}
	request, err := s.request("next action")
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

func TestModelReceivesCommandRoutingAndConciseThinkingInstructions(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	r, _, _, _, _ := realBatchRunner(t, nil)
	r.Generator = g
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[0]
	if !strings.Contains(request.Messages[0].Content, "Do not write code while thinking") || !strings.Contains(request.Messages[0].Content, "Inspect relevant source with execute_command") {
		t.Fatal("tool selection or thinking instructions were not delivered")
	}
	for _, definition := range request.Tools {
		if definition.Name != "execute_command" {
			continue
		}
		for _, text := range []string{"uv init/add/run", "cat/head/tail", "entire command", "nothing runs", "stdout, stderr, exit code"} {
			if !strings.Contains(definition.Description, text) {
				t.Fatalf("missing %q in delivered command contract", text)
			}
		}
		if request.Temperature == nil || *request.Temperature != 1 {
			t.Fatal("main agent temperature was not sent as 1")
		}
		return
	}
	t.Fatal("execute_command was not delivered")
}

// Purpose: Verify the main agent receives system, development and project instructions once.
// Pattern: Runtime prompt contract tests.
// Usage: go test ./internal/agent/run -run TestMainAgentPrompt.
// Related: internal/agent/run/run.go, internal/agent/prompt/defaults.go.
package run

import (
	"context"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"strconv"
	"strings"
	"testing"
)

func TestMainAgentPrompt(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	r := fixture(g, &executor{}, &journal{})
	templates := prompt.Defaults()
	templates.System = "COMMON_RULE"
	templates.Development = "DEVELOPMENT_RULE"
	r.Prompts, r.ProjectInstructions = &templates, "PROJECT_RULE"
	r.ProjectRoot = "/projects/selected directory"
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[0]
	if !strings.Contains(request.Messages[0].Content, "Project directory (project_root): "+strconv.Quote(r.ProjectRoot)) {
		t.Fatal("model was not told its actual project directory")
	}
	for _, rule := range []string{"COMMON_RULE", "DEVELOPMENT_RULE", "PROJECT_RULE"} {
		if strings.Count(request.Messages[0].Content, rule) != 1 {
			t.Fatalf("missing or repeated %s", rule)
		}
	}
	if request.Temperature == nil || *request.Temperature != 1 {
		t.Fatal("main agent temperature was not sent as 1")
	}
	last := request.Messages[len(request.Messages)-1].Content
	if last != "hello" {
		t.Fatal("current work context was lost")
	}
	for _, term := range []string{"COMMON_RULE", "DEVELOPMENT_RULE"} {
		if strings.Contains(last, term) {
			t.Fatalf("decision repeated system instructions: %s", term)
		}
	}
}

func TestProjectDirectorySurvivesCompaction(t *testing.T) {
	s := execution{runner: Runner{ProjectRoot: "/projects/new\nquoted\"path", Redact: func(s string) string { return s }}, limits: DefaultLimits(), system: "rules", memoryContext: "Earlier work used /projects/old", checkpoint: runrecord.Checkpoint{Objective: "Continue here"}}
	request, err := s.request()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request.Messages[0].Content, "Project directory (project_root): "+strconv.Quote(s.runner.ProjectRoot)) {
		t.Fatal("compacted history lost or corrupted the active project directory")
	}
}

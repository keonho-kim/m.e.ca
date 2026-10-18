// Purpose: Verify the main agent receives system, development and project instructions once.
// Pattern: Runtime prompt contract tests.
// Usage: go test ./internal/agent/run -run TestMainAgentPrompt.
// Related: internal/agent/run/run.go, internal/agent/prompt/defaults.go.
package run

import (
	"context"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
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
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[0]
	for _, rule := range []string{"COMMON_RULE", "DEVELOPMENT_RULE", "PROJECT_RULE"} {
		if strings.Count(request.Messages[0].Content, rule) != 1 {
			t.Fatalf("missing or repeated %s", rule)
		}
	}
	if request.Temperature == nil || *request.Temperature != 1 {
		t.Fatal("main agent temperature was not sent as 1")
	}
	last := request.Messages[len(request.Messages)-1].Content
	if !strings.Contains(last, "Work observations:") || !strings.HasSuffix(last, "Current request:\nhello") {
		t.Fatal("current work context was lost")
	}
	for _, term := range []string{"COMMON_RULE", "DEVELOPMENT_RULE"} {
		if strings.Contains(last, term) {
			t.Fatalf("decision repeated system instructions: %s", term)
		}
	}
}

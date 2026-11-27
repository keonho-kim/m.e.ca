// Purpose: Verify the main agent receives the applicable system and project instructions.
// Pattern: Runtime prompt contract tests.
// Usage: go test ./internal/agent/run -run TestMainAgentPrompt.
// Related: internal/agent/run/run.go, internal/agent/prompt/defaults.go.
package run

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	toolcatalog "github.com/keonho-kim/meca/internal/tools/local/catalog"
)

func TestMainAgentPrompt(t *testing.T) {
	g := &generator{replies: []string{"Hello"}}
	r := fixture(g, &executor{}, &journal{})
	templates := prompt.Defaults()
	templates.System = "COMMON_RULE"
	r.Prompts, r.ProjectInstructions = &templates, "PROJECT_RULE"
	r.ProjectRoot = "/projects/selected directory"
	if _, err := r.Run(context.Background(), "hello", approve, emit); err != nil {
		t.Fatal(err)
	}
	request := g.requests[0]
	if !strings.Contains(request.Messages[0].Content, "Project directory (project_root): "+strconv.Quote(r.ProjectRoot)) {
		t.Fatal("model was not told its actual project directory")
	}
	for _, rule := range []string{"COMMON_RULE", "PROJECT_RULE"} {
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
}

type browserPromptTools struct{ *executor }

func (e browserPromptTools) Definitions() []toolapi.Definition {
	return append(e.executor.Definitions(), toolcatalog.Definitions(toolapi.BrowserAction, toolapi.BrowserInspect)...)
}

func TestBrowserGuidanceFollowsToolAvailability(t *testing.T) {
	for _, profile := range []prompt.Profile{prompt.Single, prompt.Team} {
		for _, available := range []bool{false, true} {
			t.Run(string(profile)+"/installed="+strconv.FormatBool(available), func(t *testing.T) {
				g, e := &generator{replies: []string{"done"}}, &executor{}
				r := fixture(g, e, &journal{})
				profilePrompt := prompt.DefaultsFor(profile)
				r.Prompts = &profilePrompt
				r.Settings.Scope = policy.System
				if available {
					r.Tools = browserPromptTools{e}
				}
				if _, err := r.Run(t.Context(), "Inspect the project", approve, emit); err != nil {
					t.Fatal(err)
				}
				request := g.requests[0]
				system := request.Messages[0].Content
				for _, name := range []string{toolapi.BrowserAction, toolapi.BrowserInspect} {
					declared := false
					for _, definition := range request.Tools {
						declared = declared || definition.Name == "codemode" && strings.Contains(definition.Description, name+"(args:")
					}
					if declared != available || strings.Contains(system, name) != available {
						t.Errorf("browser tool and guidance have different availability for %s", name)
					}
				}
			})
		}
	}
}

func TestProjectDirectorySurvivesCompaction(t *testing.T) {
	s := execution{runner: Runner{ProjectRoot: "/projects/new\nquoted\"path", Redact: func(s string) string { return s }}, limits: DefaultLimits(), system: "rules", memoryContext: "Earlier work used /projects/old", checkpoint: runrecord.Checkpoint{Objective: "Continue here"}}
	request, err := s.request(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(request.Messages[0].Content, "Project directory (project_root): "+strconv.Quote(s.runner.ProjectRoot)) {
		t.Fatal("compacted history lost or corrupted the active project directory")
	}
}

// Purpose: Exercise autonomous skill discovery, path loading and versioned reference pagination through native calls.
// Pattern: Runtime integration test using real local tools and scripted model decisions.
// Usage: go test ./internal/agent/run -run TestSkillToolsPageReferences.
// Related: internal/tools/local/catalog/skills.go, internal/tools/local/skills.go, internal/skills/resources.go.
package run

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func skillDeliveryTools(t *testing.T) (*local.Executor, string) {
	t.Helper()
	project, home := t.TempDir(), t.TempDir()
	tools, err := local.NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s }, local.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	return tools, project
}

func lastSkillToolOutput(t *testing.T, request modelapi.Request, name string, output any) {
	t.Helper()
	for i := len(request.Messages) - 1; i >= 0; i-- {
		message := request.Messages[i]
		if message.Role == "tool" && message.ToolName == name {
			if err := json.Unmarshal([]byte(message.Content), output); err != nil {
				t.Fatal(err)
			}
			return
		}
	}
	t.Fatalf("missing %s result", name)
}

func TestSkillToolsPageReferences(t *testing.T) {
	tools, project := skillDeliveryTools(t)
	dir := filepath.Join(project, ".agents", "skills", "guide")
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: guide\ndescription: Inspect reference guidance\n---\nRead references/guide.txt before work."), 0600); err != nil {
		t.Fatal(err)
	}
	const original = "First reference. Remaining guidance."
	if err := os.WriteFile(filepath.Join(dir, "references", "guide.txt"), []byte(original), 0600); err != nil {
		t.Fatal(err)
	}
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Tools, r.ProjectRoot = tools, project
	step, id, first := 0, "", skills.Resource{}
	r.Generator = statisticsModel(func(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
		step++
		name, args := "", map[string]any{}
		switch step {
		case 1:
			name = "list_skills"
		case 2:
			var page skills.Page
			lastSkillToolOutput(t, request, "list_skills", &page)
			if len(page.Items) != 1 || page.Items[0].Name != "guide" {
				t.Fatal(page)
			}
			name, args = "load_skill", map[string]any{"path": dir}
		case 3:
			var doc instruction.Document
			lastSkillToolOutput(t, request, "load_skill", &doc)
			id = doc.ID
			if id == "" {
				t.Fatal("path load omitted its reference ID")
			}
			name, args = "read_skill_file", map[string]any{"id": id, "path": "references/guide.txt", "limit": 8}
		case 4:
			lastSkillToolOutput(t, request, "read_skill_file", &first)
			if first.NextOffset != 8 || first.Revision == "" {
				t.Fatal(first)
			}
			name, args = "read_skill_file", map[string]any{"id": id, "path": "references/guide.txt", "offset": first.NextOffset, "revision": first.Revision}
		case 5:
			var next skills.Resource
			lastSkillToolOutput(t, request, "read_skill_file", &next)
			if next.Revision != first.Revision || first.Text+next.Text != original || next.NextOffset != 0 {
				t.Fatalf("reference continuation failed: %+v", next)
			}
			return emit(modelapi.Event{Delta: "done", Done: true})
		default:
			t.Fatal("skill workflow did not finish")
		}
		body, err := json.Marshal(args)
		if err != nil {
			return err
		}
		return emit(modelapi.Event{Done: true, Calls: []toolapi.Call{{ID: "skill_step_" + strconv.Itoa(step), Name: name, Arguments: body}}})
	})
	if _, err := r.Run(t.Context(), "Review the available reference guidance", approve, emit); err != nil {
		t.Fatal(err)
	}
	if step != 5 {
		t.Fatal("incomplete skill workflow", step)
	}
}

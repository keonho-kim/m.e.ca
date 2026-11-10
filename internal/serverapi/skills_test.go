// Purpose: Verify skill previews and committed read notifications across all native transports.
// Pattern: Observable transport contract tests.
// Usage: go test ./internal/serverapi -run TestSkill.
// Related: internal/serverapi/skills.go, internal/serverapi/fixture_test.go.
package serverapi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/model"
	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
)

func serverSkill(t *testing.T, project string) {
	t.Helper()
	dir := filepath.Join(project, ".agents/skills/review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review source\ncompatibility: Requires a Git working tree\n---\nUnique-skill-instructions"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "reference.txt"), []byte("Reference text"), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestSkillReadContractsAcrossTransports(t *testing.T) {
	for _, kind := range []string{"sse", "socket", "grpc", "grpc-web", "connect"} {
		t.Run(kind, func(t *testing.T) {
			f := newFixture(t, generateFunc(func(_ context.Context, r model.Request, emit func(model.Event) error) error {
				text := ""
				for _, m := range r.Messages {
					text += m.Content
				}
				if strings.Count(text, "Unique-skill-instructions") != 1 {
					t.Errorf("skill instructions missing or duplicated: %s", text)
				}
				return emit(model.Event{Delta: "Done", Done: true})
			}), false)
			serverSkill(t, f.project)
			ctx, w := deadline(t), tlsWire(t, kind, f.api)
			suggestions := call(t, ctx, w, &v1.Command{Operation: &v1.Command_SuggestSkills{SuggestSkills: &v1.SkillPrefix{Prefix: "rev"}}}).GetSkills()
			if len(suggestions.GetItems()) != 1 || suggestions.Items[0].Name != "review" {
				t.Fatal(suggestions)
			}
			page := call(t, ctx, w, &v1.Command{Operation: &v1.Command_ListSkills{ListSkills: &v1.SkillListRequest{}}}).GetSkills()
			if len(page.GetItems()) != 1 || page.Items[0].Name != "review" || page.Items[0].Compatibility != "Requires a Git working tree" {
				t.Fatal(page)
			}
			preview := call(t, ctx, w, &v1.Command{Operation: &v1.Command_GetSkill{GetSkill: &v1.SkillRequest{Id: page.Items[0].Id}}}).GetSkill()
			if preview.Body != "Unique-skill-instructions" || preview.Revision == "" || preview.Skill.Compatibility != "Requires a Git working tree" || f.executed.Load() != 0 {
				t.Fatal(preview)
			}
			file := call(t, ctx, w, &v1.Command{Operation: &v1.Command_ReadSkillFile{ReadSkillFile: &v1.SkillFileRequest{Id: page.Items[0].Id, Path: "reference.txt"}}}).GetSkillFile()
			if file.Text != "Reference text" || file.Revision == "" {
				t.Fatal(file)
			}
			continued := call(t, ctx, w, &v1.Command{Operation: &v1.Command_ReadSkillFile{ReadSkillFile: &v1.SkillFileRequest{Id: page.Items[0].Id, Path: "reference.txt", Offset: 4, Revision: file.Revision}}}).GetSkillFile()
			if continued.Text != file.Text[4:] || continued.Revision != file.Revision {
				t.Fatal("resource revision lost across transport", continued)
			}
			bad, err := w.call(ctx, &v1.Command{Operation: &v1.Command_ReadSkillFile{ReadSkillFile: &v1.SkillFileRequest{Id: page.Items[0].Id, Path: "../SKILL.md"}}})
			if err != nil || bad.GetError().GetCode() != "invalid_argument" {
				t.Fatal(bad, err)
			}
			state := call(t, ctx, w, &v1.Command{Operation: &v1.Command_SelectSession{SelectSession: &v1.SelectSession{Create: true}}}).GetState()
			call(t, ctx, w, &v1.Command{Operation: &v1.Command_Start{Start: &v1.StartRequest{SessionId: state.Session.Id, Key: "skill-request", Text: "Use #review"}}})
			watch, err := w.watch(ctx, state.Cursor)
			if err != nil {
				t.Fatal(err)
			}
			defer watch.close()
			reads := 0
			for {
				n := next(t, watch)
				if e := n.GetEvent(); e != nil {
					if e.Kind == "skill_loaded" {
						reads++
						if e.Skill.GetName() != "review" || e.OccurredAt == "" || e.RunId == "" || strings.Contains(e.Text, "Unique-skill-instructions") {
							t.Fatal(e)
						}
					}
					if e.Kind == "request_finished" {
						break
					}
				}
			}
			if reads != 1 {
				t.Fatalf("reads=%d", reads)
			}
			history := call(t, ctx, w, &v1.Command{Operation: &v1.Command_History{History: &v1.HistoryRequest{Limit: 100}}}).GetHistory()
			found := false
			for _, item := range history.Items {
				found = found || item.GetEvent().GetSkill().GetName() == "review"
			}
			if !found {
				t.Fatal("skill read missing from reconnect history", history)
			}
		})
	}
}

// Purpose: Verify explicit skill reads precede model work and are published only after persistence.
// Pattern: Request behavior tests with local fixtures.
// Usage: go test ./internal/agent/run -run TestSkill.
// Related: internal/agent/run/skills.go, internal/agent/run/conversation.go.
package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/skills"
)

func skillRunner(t *testing.T, j *journal) (Runner, *generator) {
	t.Helper()
	project := t.TempDir()
	dir := filepath.Join(project, ".agents", "skills", "review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review source\n---\nRead-me-exactly"), 0600); err != nil {
		t.Fatal(err)
	}
	g := &generator{replies: []string{"Done"}}
	r := fixture(g, &executor{}, j)
	r.Skills = skills.New(project, t.TempDir(), skills.Options{Redact: r.Redact, Check: func(string) error { return nil }})
	return r, g
}

func TestSkillMentionReadBeforeFirstModelRequest(t *testing.T) {
	j := &journal{}
	r, g := skillRunner(t, j)
	var reads int
	_, err := r.Run(context.Background(), "Use #review", approve, func(e runrecord.Event) error {
		if e.Skill != nil {
			reads++
			if len(j.messages) < 2 || len(j.events) == 0 {
				t.Fatal("read published before commit")
			}
		}
		return nil
	})
	if err != nil || reads != 1 {
		t.Fatalf("reads=%d err=%v", reads, err)
	}
	text := ""
	for _, m := range g.requests[0].Messages {
		text += m.Content
	}
	if strings.Count(text, "Read-me-exactly") != 1 {
		t.Fatalf("request=%s", text)
	}
}

func TestSkillFailedCommitDoesNotPublishRead(t *testing.T) {
	j := &journal{failKind: "skill_loaded"}
	r, g := skillRunner(t, j)
	reads := 0
	_, err := r.Run(context.Background(), "#review", approve, func(e runrecord.Event) error {
		if e.Skill != nil {
			reads++
		}
		return nil
	})
	if err == nil || reads != 0 || len(g.requests) != 0 {
		t.Fatalf("reads=%d requests=%d err=%v", reads, len(g.requests), err)
	}
}

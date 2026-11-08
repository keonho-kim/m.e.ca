// Purpose: Verify an autonomous ordinary skill read reaches the next model decision exactly once.
// Pattern: Runtime integration test using the real local read adapter.
// Usage: go test ./internal/agent/run -run TestSkillTool.
// Related: internal/agent/run/action_result.go, internal/tools/local/skills.go.
package run

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tools/local"
)

func TestSkillToolReadReachesNextDecisionWithoutBodyNotifications(t *testing.T) {
	project, home := t.TempDir(), t.TempDir()
	dir := filepath.Join(project, ".agents/skills/review")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: review\ndescription: Review source\n---\nAutonomous-read-instruction"), 0600); err != nil {
		t.Fatal(err)
	}
	tools, err := local.NewExecutor(project, home, filepath.Join(home, ".meca"), func(s string) string { return s }, local.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := tools.Close(); err != nil {
			t.Error(err)
		}
	})
	g := &generator{replies: []string{nativeCall("load_skill", map[string]any{"id": "project:review"}), "Done"}}
	r := fixture(g, &executor{}, &journal{})
	r.Tools = tools
	reads := 0
	_, err = r.Run(context.Background(), "Review the source", approve, func(e runrecord.Event) error {
		if e.Skill != nil {
			reads++
			if strings.Contains(e.Text, "Autonomous-read-instruction") {
				t.Error("instruction body leaked into notification text")
			}
		}
		return nil
	})
	if err != nil || reads != 1 {
		t.Fatalf("reads=%d err=%v", reads, err)
	}
	text := ""
	for _, m := range g.requests[1].Messages {
		text += m.Content
	}
	if strings.Count(text, "Autonomous-read-instruction") != 1 {
		t.Fatal("read missing or duplicated in the next decision")
	}
}

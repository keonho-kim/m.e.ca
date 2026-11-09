// Purpose: Check superseded skill versions after context compaction.
// Pattern: Request-state regression test.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/skills.go.
package run

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestSupersededSkillRetention(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s, release, err := r.start(t.Context(), "Review", emit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	for i := 0; i < 4; i++ {
		doc := instruction.Document{Metadata: instruction.Metadata{ID: "project:review", Name: "review", Source: "project", Description: "Review"}, Revision: fmt.Sprintf("%064x", i), Body: fmt.Sprintf("instruction-version-%d", i)}
		encoded, _ := json.Marshal(doc)
		if err := s.acceptResult(t.Context(), validatedCall{call: toolapi.Call{ID: fmt.Sprintf("c%d", i), Name: "load_skill"}}, toolapi.Result{Content: encoded, Skill: &doc}, nil); err != nil {
			t.Fatal(err)
		}
	}
	s.messages = nil
	s.memoryContext = "Old source history was compacted"
	req, err := s.request()
	if err != nil {
		t.Fatal(err)
	}
	var all strings.Builder
	for _, m := range req.Messages {
		all.WriteString(m.Content)
	}
	if strings.Contains(all.String(), "instruction-version-0") {
		t.Fatalf("superseded versions reinjected; retained=%d", len(s.skillReads))
	}
}

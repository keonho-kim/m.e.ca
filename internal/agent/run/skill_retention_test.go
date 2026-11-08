// Purpose: Prevent duplicate retained skill instructions.
// Pattern: Resource ownership regression test.
// Usage: go test ./internal/agent/run.
// Related: internal/agent/run/action_result.go.
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

func TestRepeatedSkillRevisionRetainedOnce(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s, release, err := r.start(t.Context(), "Review", emit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	document := instruction.Document{Metadata: instruction.Metadata{ID: "project:review", Name: "review", Source: "project", Description: "Review"}, Revision: strings.Repeat("a", 64), Body: "unique-retained-instruction"}
	encoded, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 4; i++ {
		call := validatedCall{call: toolapi.Call{ID: fmt.Sprintf("call_%d", i), Name: "load_skill"}}
		if err := s.acceptResult(t.Context(), call, toolapi.Result{Content: encoded, Skill: &document}, nil); err != nil {
			t.Fatal(err)
		}
	}
	s.messages = nil
	s.memoryContext = "Older history was compacted"
	request, err := s.request()
	if err != nil {
		t.Fatal(err)
	}
	copies := 0
	for _, m := range request.Messages {
		copies += strings.Count(m.Content, document.Body)
	}
	if copies != 1 {
		t.Fatalf("same revision retained %d times after original history left context; retained reads=%d", copies, len(s.skillReads))
	}
}

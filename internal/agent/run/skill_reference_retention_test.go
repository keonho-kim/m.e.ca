// Purpose: Preserve reference pages through compaction and reject over-budget reads before successful publication.
// Pattern: Request-state behavior regression tests.
// Usage: go test ./internal/agent/run -run 'TestSkillReference|TestSkillRetentionBudget'.
// Related: internal/agent/run/skill_retention.go, internal/agent/run/execution_result.go.
package run

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func TestSkillReferenceSurvivesCompactionAndReplacesChangedPages(t *testing.T) {
	r := fixture(&generator{}, &executor{}, &journal{})
	s, release, err := r.start(t.Context(), "Review", emit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	read := func(version, body string, offset int64) {
		t.Helper()
		ref := &instruction.ReferenceRead{ID: "project:review", Path: "references/guide.md", Revision: version, Offset: offset}
		encoded, _ := json.Marshal(skills.Resource{Revision: version, Text: body, Offset: offset})
		call := toolapi.Call{ID: "call_read", Name: "read_skill_file"}
		if err := s.acceptResult(t.Context(), validatedCall{call: call}, toolapi.Result{Content: encoded, SkillFile: ref}, nil); err != nil {
			t.Fatal(err)
		}
	}
	read("one", "old-page-one", 0)
	read("one", "old-page-two", 64)
	read("two", "current-reference-canary", 0)
	s.messages = nil
	s.memoryContext = "Earlier records were compacted"
	request, err := s.request()
	if err != nil {
		t.Fatal(err)
	}
	var context strings.Builder
	for _, m := range request.Messages {
		context.WriteString(m.Content)
	}
	if strings.Contains(context.String(), "old-page-") || strings.Count(context.String(), "current-reference-canary") != 1 || len(s.skillReads) != 1 {
		t.Fatal("reference revision lost or stale pages retained")
	}
}

func TestSkillRetentionBudgetIsNotAPersistenceFailure(t *testing.T) {
	j := &journal{}
	r := fixture(&generator{}, &executor{}, j)
	s, release, err := r.start(t.Context(), "Review", emit, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	if err := s.retainInstruction("skill:existing", "one", modelapi.Message{Content: strings.Repeat("x", maxSkillReadBytes)}); err != nil {
		t.Fatal(err)
	}
	call := toolapi.Call{ID: "call_read", Name: "read_skill_file", Arguments: json.RawMessage(`{"id":"project:review","path":"guide.md"}`)}
	result := toolapi.Result{Content: json.RawMessage(`{"text":"new-instruction"}`), SkillFile: &instruction.ReferenceRead{ID: "project:review", Path: "guide.md", Revision: "one"}}
	result, err = s.recordExecution(t.Context(), validatedCall{call: call}, &readyExecution{call: call}, result, nil, emit)
	if errors.Is(err, errRuntimeRecord) || faults.Code(err) != diagcatalog.ResourceExhausted || !result.Failed {
		t.Fatal("capacity misclassified", result, err)
	}
	if len(j.events) == 0 || len(s.skillReads) != 1 {
		t.Fatal("failure not committed or instructions changed")
	}
	for _, event := range j.events {
		if event.Kind == "tool_succeeded" {
			t.Fatal("over-budget read published as success")
		}
	}
}

// Purpose: Hide execution provenance while retaining current files and scheduling information.
// Pattern: Model-context projection regression test.
// Usage: go test ./internal/agent/run -run TestContextOmits.
// Related: internal/agent/run/task_context.go, internal/agent/tool/facts.go.
package run

import (
	"strings"
	"testing"

	"github.com/keonho-kim/v-code/internal/agent/completion"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
)

func TestContextOmitsFileProvenance(t *testing.T) {
	s := execution{checkpoint: Checkpoint{Work: completion.State{CurrentFiles: []toolapi.FileFact{{Path: "/p/a", Version: "PRIVATE_DIGEST", SourceID: "PRIVATE_SOURCE", ObservedRun: "PRIVATE_RUN", Revision: 12345, Observation: "present"}}}}}
	body, err := s.taskContext()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "PRIVATE_") || strings.Contains(string(body), "12345") {
		t.Fatalf("provenance leaked: %s", body)
	}
	if !strings.Contains(string(body), "/p/a") || !strings.Contains(string(body), "present") {
		t.Fatalf("current state missing: %s", body)
	}
}

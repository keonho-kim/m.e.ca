// Purpose: Verify refined-plan events reject unusable text at the persistence boundary.
// Pattern: Event value contract tests.
// Usage: go test ./internal/agent/record -run TestPlan.
// Related: internal/agent/record/event_validation.go.
package record

import (
	"strings"
	"testing"
)

func TestPlanEventRequiresBoundedText(t *testing.T) {
	for _, text := range []string{"", " \n\t", string([]byte{0xff}), "plan\x00hidden", strings.Repeat("x", 16*1024*1024+1)} {
		if err := (Event{Kind: "lat_plan", Text: text}).Check(); err == nil {
			t.Errorf("invalid plan accepted: bytes=%d", len(text))
		}
	}
	if err := (Event{Kind: "lat_plan", Text: "Inspect source, implement and verify."}).Check(); err != nil {
		t.Fatal(err)
	}
}

// Purpose: Verify refined plans enter context as proposed implementation guidance rather than completed work.
// Pattern: Context projection contract test.
// Usage: go test ./internal/agent/lat -run TestPlan.
// Related: internal/agent/lat/plan.go.
package lat

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPlanKeepsQuotedImplementationGuidance(t *testing.T) {
	text := "Inspect \"api.go\"\nImplement handler."
	message := Plan(text)
	quoted, _ := json.Marshal(text)
	if message.Role != "system" || !strings.Contains(message.Content, string(quoted)) || !strings.Contains(message.Content, "implementation plan") || !strings.Contains(message.Content, "not completed work") || !strings.Contains(message.Content, "Current and later user directions take precedence") || !strings.Contains(message.Content, "not completed work or new requirements") {
		t.Fatalf("plan promoted to observed outcome: %+v", message)
	}
}

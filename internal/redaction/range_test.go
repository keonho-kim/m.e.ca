// Purpose: Verify reference pages cannot expose credentials crossing either byte boundary.
// Pattern: Pure redaction contract tests.
// Usage: go test ./internal/redaction -run TestRange.
// Related: internal/redaction/range.go, internal/skills/resources.go.
package redaction

import (
	"strings"
	"testing"
)

func TestRangeRedactionMasksBothBoundaries(t *testing.T) {
	var registry Registry
	if err := registry.Remember("credential-canary"); err != nil {
		t.Fatal(err)
	}
	text := "before credential-canary after"
	for _, range_ := range [][2]int{{10, 19}, {0, 12}, {16, len(text)}} {
		got, err := registry.RedactRange(text, range_[0], range_[1])
		if err != nil || !strings.Contains(got, "[redacted]") || strings.Contains(got, "canary") || strings.Contains(got, "credential") {
			t.Fatal(got, err)
		}
	}
}

// Purpose: Own the shared free-form reason contract for native intent and internal review requests.
// Pattern: Pure boundary policy.
// Usage: Tool parsing, grounding and review parsers call CheckReason without truncating source values.
// Related: internal/agent/tool/intent.go, internal/agent/grounding/record.go, internal/agent/conclave/verdict.go.
package tool

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const MaxReasonBytes = 32 * 1024 * 1024

func CheckReason(reason string) error {
	if strings.TrimSpace(reason) == "" || len(reason) > MaxReasonBytes || !utf8.ValidString(reason) || strings.ContainsFunc(reason, func(r rune) bool { return unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' }) {
		return fmt.Errorf("reason must be nonempty UTF-8 prose within 32 MiB, without terminal control characters")
	}
	return nil
}

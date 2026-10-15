// Purpose: Validate stored summary bounds and expose the continuation memo as context.
// Pattern: Simple Module.
// Usage: Runtime requests context and persistence validates memory revisions.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

import (
	"fmt"
	"unicode/utf8"
)

func (m Memory) Context() (string, error) {
	if m.Revision == 0 {
		return "", nil
	}
	return m.Markdown, nil
}

func (m Memory) Check(maxSummary int) error {
	if m.Revision < 0 || m.Through < 0 || m.UsageThrough < 0 || len(m.Markdown) > maxSummary || !utf8.ValidString(m.Markdown) {
		return fmt.Errorf("invalid compaction memory")
	}
	return nil
}

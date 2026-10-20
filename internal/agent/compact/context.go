// Purpose: Validate stored continuation memory bounds and text encoding.
// Pattern: Simple Module.
// Usage: The compaction controller and persistence check memory revisions before use.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

import (
	"fmt"
	"unicode/utf8"
)

func (m Memory) Check(maxSummary int) error {
	if m.Revision < 0 || m.Through < 0 || m.UsageThrough < 0 || len(m.Markdown) > maxSummary || !utf8.ValidString(m.Markdown) {
		return fmt.Errorf("invalid compaction memory")
	}
	return nil
}

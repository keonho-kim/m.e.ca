// Purpose: Validate stored continuation memory bounds and text encoding.
// Pattern: Simple Module.
// Usage: The compaction controller and persistence check memory revisions before use.
// Related: internal/agent/compact/types.go, internal/agent/compact/controller.go.
package compact

import (
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (m Memory) Check(maxSummary int) error {
	if m.Revision < 0 || m.Through < 0 || m.UsageThrough < 0 || len(m.Markdown) > maxSummary || !utf8.ValidString(m.Markdown) {
		return faults.Errorf(diagcatalog.CompactInvalidCompactionMemory)
	}
	return nil
}

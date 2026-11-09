// Purpose: Interpret model finish reasons without losing output-limit failures.
// Pattern: Simple Module.
// Usage: Protocol stream decoders use sanitizeReason.
// Related: internal/provider/protocol/chunk.go.
package protocol

import (
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func sanitizeReason(reason string) error {
	if reason == "length" {
		return &modelapi.OutputLimitError{Reason: "output token limit"}
	}
	if reason == "content_filter" || reason == "error" || reason == "abort" {
		return faults.Errorf(diagcatalog.ProtocolModelResponseStopped, strings.ReplaceAll(reason, "\n", ""))
	}
	return nil
}

// Purpose: Interpret model finish reasons without losing output-limit failures.
// Pattern: Simple Module.
// Usage: Protocol stream decoders use sanitizeReason.
// Related: internal/provider/protocol/chunk.go.
package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"strings"
)

func sanitizeReason(reason string) error {
	if reason == "length" {
		return &modelapi.OutputLimitError{Reason: "output token limit"}
	}
	if reason == "content_filter" || reason == "error" || reason == "abort" {
		return fmt.Errorf("Model response stopped: %s", strings.ReplaceAll(reason, "\n", ""))
	}
	return nil
}

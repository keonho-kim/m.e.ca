// Purpose: Interpret the terminal report of a completed read-only review session.
// Pattern: Model response boundary.
// Usage: The native run owner calls Parse after its normal final-response transition.
// Related: internal/agent/lat/packet.go, internal/agent/run/lat.go.
package lat

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Result struct {
	Feedback string
	Clean    bool
}

const maxResponseBytes = 16 * 1024 * 1024

func Parse(text string) (Result, error) {
	if len(text) > maxResponseBytes {
		return Result{}, fmt.Errorf("LAT review response exceeds %d bytes", maxResponseBytes)
	}
	feedback := strings.TrimSpace(text)
	if feedback == "" || !utf8.ValidString(feedback) {
		return Result{}, fmt.Errorf("LAT review did not produce a complete response")
	}
	return Result{Feedback: feedback, Clean: feedback == "NO_FINDINGS"}, nil
}

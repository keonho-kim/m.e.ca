// Purpose: Prevent copied read-display labels from becoming source text by default.
// Pattern: Pure candidate guard with an explicit literal-content escape.
// Usage: Candidate preparation checks supplied text before format normalization and review.
// Related: internal/tools/local/candidate.go, internal/tools/local/anchors/store.go.
package textformat

import (
	"fmt"
	"regexp"
)

var readLabel = regexp.MustCompile(`(?m)^(?:[1-9][0-9]*)?#[0-9a-f]{8} \| `)

func CheckAnnotations(before, after []byte, policy string) error {
	if policy == "literal" || !readLabel.Match(after) || readLabel.Match(before) {
		return nil
	}
	return fmt.Errorf("read_display_annotation: replacement contains line/hash display labels; copy only source text after the separator, or use format_policy=literal when these labels are intentionally part of the requested file")
}

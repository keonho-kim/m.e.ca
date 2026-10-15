// Purpose: Prevent copied read-display labels from becoming source text by default.
// Pattern: Pure candidate guard with an explicit literal-content escape.
// Usage: Candidate preparation checks supplied text before format normalization and review.
// Related: internal/tools/local/candidate.go, internal/tools/local/anchors/store.go.
package textformat

import (
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"regexp"
)

var readLabel = regexp.MustCompile(`(?m)^(?:[1-9][0-9]*)?#[0-9a-f]{8} \| `)

func CheckAnnotations(before, after []byte, policy string) error {
	if policy == "literal" || !readLabel.Match(after) || readLabel.Match(before) {
		return nil
	}
	return &toolapi.Failure{Code: toolapi.InvalidArgument, Detail: "read_display_annotation: replacement contains read_file display labels", Next: "Keep line#hash labels only in start_anchor/end_anchor. In new_text copy only the source text after the separator; keep format_policy=preserve. Existing valid anchors can be reused."}
}

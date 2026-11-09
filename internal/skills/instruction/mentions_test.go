// Purpose: Verify explicit skill mentions ignore quoted, escaped and historical-looking syntax.
// Pattern: Parser behavior tests.
// Usage: go test ./internal/skills/instruction -run TestMentions.
// Related: internal/skills/instruction/mentions.go.
package instruction

import (
	"reflect"
	"testing"
)

func TestMentions(t *testing.T) {
	input := "Use #review then #test-suite and #review. Ignore `#inline`, " + `\#escaped` + ", #123, https://x/#url, \"#quoted\".\n> #quoted-block\n```go\n#fenced\n```\n#last"
	if got := Mentions(input); !reflect.DeepEqual(got, []string{"review", "test-suite", "last"}) {
		t.Fatalf("mentions=%q", got)
	}
}

func TestMentionsQualifiedNamesAndNaturalApostrophes(t *testing.T) {
	input := "Don't skip #meca:review or #shared:123. Ignore ``#code`` and #review_UPPER.\n    #indented\n　#unicode-space"
	if got := Mentions(input); !reflect.DeepEqual(got, []string{"meca:review", "shared:123", "unicode-space"}) {
		t.Fatalf("mentions=%q", got)
	}
}

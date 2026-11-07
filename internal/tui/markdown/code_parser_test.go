// Purpose: Verify compact code bodies preserve the native parser's indentation, EOF and closing rules.
// Pattern: Differential third-party parser boundary tests.
// Usage: go test ./internal/tui/markdown -run TestCompactCode.
// Related: internal/tui/markdown/code_parser.go, internal/tui/markdown/parser.go.
package markdown

import (
	"reflect"
	"testing"

	"github.com/yuin/goldmark/v2/parser"
)

func TestCompactCodeMatchesNativeParser(t *testing.T) {
	native := parser.New()
	for _, source := range []string{
		"```text\nbody\n\n```\n", "```\nbody", "```\n\n\n```",
		"```text\r\nbody\r\n```\r\n", "  ```\n  body\n  ```\n",
		"- ```\n  first\n\n  second\n  ```\n", "> ```\n> body\n> ```\n",
		"    first\n\n    second\n\n", "\tfirst\n\t\tsecond\n", "    EOF",
		"```\nfirst\n```\n\n    second\n", "```\n\ttext\n```\n",
	} {
		bytes := []byte(source)
		want := renderer{source: bytes, width: 80}
		want.blocks(native.Parse(bytes), "", 0)
		got := renderer{source: bytes, width: 80}
		got.blocks(documentParser.Parse(bytes), "", 0)
		if !reflect.DeepEqual(got.rows, want.rows) {
			t.Fatalf("code semantics changed for %q", source)
		}
	}
}

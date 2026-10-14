// Purpose: Verify trimmed output releases references to removed text beyond the live slice.
// Pattern: Memory ownership regression test.
// Usage: go test ./internal/tui/viewport -run TestTrimReleasesText.
// Related: internal/tui/viewport/viewport.go.
package viewport

import "testing"

func TestTrimReleasesText(t *testing.T) {
	v := View{Blocks: make([]Block, 1001), Follow: true}
	for i := range v.Blocks {
		v.Blocks[i] = Block{ID: i, Text: "held text"}
	}
	v.bound(true)
	retained := v.Blocks[:cap(v.Blocks)]
	for _, b := range retained[len(v.Blocks):] {
		if b.Text != "" {
			t.Fatal("removed output still retains text")
		}
	}
}

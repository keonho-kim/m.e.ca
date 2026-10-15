// Purpose: Verify exact JSON wrappers are normalized without extracting arbitrary prose.
// Pattern: Pure boundary tests.
// Usage: go test ./internal/agent/reviewjson.
// Related: internal/agent/reviewjson/object.go.
package reviewjson

import "testing"

func TestObjectWrappers(t *testing.T) {
	for _, text := range []string{"  {\"decision\":\"pass\"} \n", "```json\r\n{\"decision\":\"pass\"}\r\n```"} {
		got, err := Object(text)
		if err != nil || got != "{\"decision\":\"pass\"}" {
			t.Fatalf("%q %v", got, err)
		}
	}
	for _, text := range []string{"Here is JSON: {}", "```json\n{}\n```\nmore", "{} {}"} {
		if _, err := Object(text); err == nil {
			t.Fatalf("accepted %q", text)
		}
	}
}

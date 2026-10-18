// Purpose: Verify file format preservation without trimming meaningful whitespace.
// Pattern: Pure byte contract tests.
// Usage: go test ./internal/tools/local/textformat.
// Related: internal/tools/local/textformat/format.go.
package textformat

import "testing"

func TestPreserve(t *testing.T) {
	for _, tc := range []struct {
		before, after, policy, want string
		fail                        bool
	}{
		{"old\n", "new", "", "new\n", false},
		{"old\r\n", "new", "", "new\r\n", false},
		{"old", "new", "", "new", false},
		{"old", "new\n", "", "", true},
		{"old\n", "new", "literal", "new", false},
		{"old\n", "new\n\n", "", "new\n\n", false},
		{"old\n", "", "", "", false},
		{"", "new", "", "new", false},
		{"old\r\n", "new\n", "", "", true},
		{"old\r\n", "new\n", "literal", "new\n", false},
	} {
		got, err := Preserve([]byte(tc.before), []byte(tc.after), tc.policy)
		if (err != nil) != tc.fail || !tc.fail && string(got) != tc.want {
			t.Fatalf("%q -> %q (%s): %q %v", tc.before, tc.after, tc.policy, got, err)
		}
	}
}

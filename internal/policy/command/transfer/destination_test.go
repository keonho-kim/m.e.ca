// Purpose: Verify local API recognition and sanitized outbound destinations without DNS.
// Pattern: Pure URL behavior tests.
// Usage: go test ./internal/policy/command/transfer -run TestDestination.
// Related: internal/policy/command/transfer/destination.go.
package transfer

import "testing"

func TestDestinationLocalAndDisplay(t *testing.T) {
	for _, value := range []string{"http://127.0.0.1:8080/posts", "http://[::1]:8080/posts", "http://localhost:8080/posts"} {
		if !LoopbackHTTP(value) {
			t.Errorf("local API rejected: %s", value)
		}
	}
	for _, value := range []string{"http://localhost.evil.test", "http://169.254.169.254/", "https://example.test/", "ssh://localhost/", "http://127.0.0.1@elsewhere.test/"} {
		if LoopbackHTTP(value) {
			t.Errorf("non-local API accepted: %s", value)
		}
	}
	if got := Display("https://user:password@host/path?secret=value#fragment"); got != "https://host/path" {
		t.Fatalf("unsafe audit destination: %q", got)
	}
	if got := Display("not a URL"); got != "" {
		t.Fatalf("invented an unresolved destination: %q", got)
	}
}

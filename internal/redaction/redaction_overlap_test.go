// Purpose: Keep overlapping and replacement-shaped credentials private in complete values.
// Pattern: Secret sanitization regression tests.
// Usage: go test ./internal/config/credentials -run TestRedactionOverlapping.
// Related: internal/redaction/redaction.go, internal/redaction/stream.go.
package redaction

import "testing"

func TestRedactionOverlappingCredentials(t *testing.T) {
	v := &Registry{}
	v.Remember("audit-key")
	v.Remember("audit-key-longer")
	v.Remember("redacted")
	if got := v.Redact("audit-key-longer! audit-key"); got != "[redacted]! [redacted]" {
		t.Fatal("credential suffix or replacement marker was exposed", got)
	}
}

func TestRedactionControlsCannotReconstructCredential(t *testing.T) {
	v := &Registry{}
	if err := v.Remember("private-key"); err != nil {
		t.Fatal(err)
	}
	text := "pri\x00vate-\x1bkey!"
	mask := v.NewStreamMask()
	for at := 1; at <= len(text); at++ {
		if got := mask(text[:at], false); got == "private-key!" {
			t.Fatal("reconstructed credential exposed")
		}
	}
	if got := mask(text, true); got != "[redacted]!" || got != v.Redact(text) {
		t.Fatal(got)
	}
}

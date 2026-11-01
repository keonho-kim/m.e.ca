// Purpose: Verify completed LAT terminal reports retain feedback and enforce bounded valid text.
// Pattern: Pure response boundary tests.
// Usage: go test ./internal/agent/lat -run TestParse.
// Related: internal/agent/lat/response.go, internal/agent/run/lat.go.
package lat

import (
	"strings"
	"testing"
)

func TestParseTerminalReport(t *testing.T) {
	for _, test := range []struct {
		name, text, feedback string
		clean                bool
	}{
		{"clean", "NO_FINDINGS", "NO_FINDINGS", true},
		{"trimmed", " \nNO_FINDINGS\t", "NO_FINDINGS", true},
		{"actionable", "\n- api.go: missing route; add the requested endpoint.\n", "- api.go: missing route; add the requested endpoint.", false},
		{"unicode", "구현 결과와 계획이 일치하지 않습니다.", "구현 결과와 계획이 일치하지 않습니다.", false},
		{"mixed", "NO_FINDINGS\nBut an endpoint is missing.", "NO_FINDINGS\nBut an endpoint is missing.", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Parse(test.text)
			if err != nil || result.Clean != test.clean || result.Feedback != test.feedback {
				t.Fatalf("parse: %+v %v", result, err)
			}
		})
	}
}

func TestParseRejectsEmptyOrInvalidText(t *testing.T) {
	for _, text := range []string{"", " \t\r\n", "\xff", "finding\n\xfe"} {
		result, err := Parse(text)
		if err == nil || result.Feedback != "" || result.Clean {
			t.Fatalf("invalid terminal report accepted: %+v %v", result, err)
		}
	}
}

func TestParseTerminalReportByteLimit(t *testing.T) {
	const limit = 16 * 1024 * 1024
	text := strings.Repeat("x", limit)
	result, err := Parse(text)
	if err != nil || len(result.Feedback) != limit || result.Clean {
		t.Fatalf("exact byte limit rejected: %d %v", len(result.Feedback), err)
	}
	for _, text := range []string{text + "x", strings.Repeat(" ", limit+1), strings.Repeat("가", limit/3+1)} {
		result, err := Parse(text)
		if err == nil || result.Feedback != "" || result.Clean {
			t.Fatalf("oversized terminal report accepted: %d %v", len(text), err)
		}
	}
}

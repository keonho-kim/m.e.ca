// Purpose: Verify running duration uses hours, minutes and seconds with spaced colons.
// Pattern: Pure display contract tests.
// Usage: go test ./internal/tui/view -run TestActivityElapsed.
// Related: internal/tui/view/activity.go.
package view

import (
	"testing"
	"time"
)

func TestActivityElapsed(t *testing.T) {
	for _, tc := range []struct {
		elapsed time.Duration
		want    string
	}{{0, "00 : 00 : 00"}, {time.Hour + 2*time.Minute + 3*time.Second, "01 : 02 : 03"}, {25 * time.Hour, "25 : 00 : 00"}} {
		if got := (Activity{Running: true, Elapsed: tc.elapsed}).Label(); got != "상태 작업 중 "+tc.want {
			t.Fatal(got)
		}
	}
	if got := (Activity{Elapsed: time.Hour}).Label(); got != "상태 대기" {
		t.Fatal(got)
	}
}

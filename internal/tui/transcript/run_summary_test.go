// Purpose: Verify archived request measurements become a muted, centered completion line.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestRunSummary.
// Related: internal/tui/transcript/events.go, internal/tui/viewport/viewport.go.
package transcript

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func TestRunSummaryFromHistory(t *testing.T) {
	var event run.Event
	if err := json.Unmarshal([]byte(`{"kind":"run_checkpoint","checkpoint":{"phase":"finished","status":"answered"},"statistics":{"elapsed_millis":3723000,"generation_millis":4000,"usage":{"input":120,"output":20}}}`), &event); err != nil {
		t.Fatal(err)
	}
	formatter := Formatter{Language: locale.English, Redact: func(s string) string { return s }}
	blocks := formatter.Event(event)
	if len(blocks) != 1 {
		t.Fatalf("completion measurements missing: %+v", blocks)
	}
	want := "Worked for 01:02:03 | INPUT 120 | OUTPUT 20 | TPS 5.0"
	if strings.TrimSpace(blocks[0].Text) != want || !blocks[0].Thinking {
		t.Fatalf("summary: %+v", blocks[0])
	}
	lines := blocks[0].Lines(100)
	if lines[0].Text != strings.Repeat(" ", (100-uniseg.StringWidth(want))/2)+want {
		t.Fatalf("summary is not centered: %q", lines[0].Text)
	}
}

func TestRunSummaryUnavailableAndZeroUsage(t *testing.T) {
	f := Formatter{Language: locale.English}
	if text := f.runSummary(run.Statistics{ElapsedMillis: 1000}).Text; !strings.Contains(text, "INPUT — | OUTPUT — | TPS —") {
		t.Fatal(text)
	}
	zero := run.Statistics{ElapsedMillis: 1000, GenerationMillis: 1000, Usage: &modelapi.TokenUsage{}}
	if text := f.runSummary(zero).Text; !strings.Contains(text, "INPUT 0 | OUTPUT 0 | TPS 0.0") {
		t.Fatal(text)
	}
	f.Language = locale.Korean
	if text := f.runSummary(zero).Text; !strings.Contains(text, "작업 시간 00:00:01 | 입력 0 | 출력 0 | 초당 토큰 0.0") {
		t.Fatal(text)
	}
}

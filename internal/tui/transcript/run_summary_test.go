// Purpose: Verify archived request measurements become a muted, centered completion line.
// Pattern: Presentation contract tests.
// Usage: go test ./internal/tui/transcript -run TestRunSummary.
// Related: internal/tui/transcript/events.go, internal/tui/viewport/viewport.go.
package transcript

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/locale"
	"github.com/rivo/uniseg"
)

func TestRunSummaryFromHistory(t *testing.T) {
	var event runrecord.Event
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
	lines := blocks[0].Window(100, 0, blocks[0].Count(100))
	if lines[0].Text != strings.Repeat(" ", (100-uniseg.StringWidth(want))/2)+want {
		t.Fatalf("summary is not centered: %q", lines[0].Text)
	}
}

func TestRunSummaryUnavailableAndZeroUsage(t *testing.T) {
	f := Formatter{Language: locale.English}
	if text := f.runSummary(runrecord.Statistics{ElapsedMillis: 1000}).Text; !strings.Contains(text, "INPUT — | OUTPUT — | TPS —") {
		t.Fatal(text)
	}
	zero := runrecord.Statistics{ElapsedMillis: 1000, GenerationMillis: 1000, Usage: &modelapi.TokenUsage{}}
	if text := f.runSummary(zero).Text; !strings.Contains(text, "INPUT 0 | OUTPUT 0 | TPS 0.0") {
		t.Fatal(text)
	}
	f.Language = locale.Korean
	if text := f.runSummary(zero).Text; !strings.Contains(text, "작업 시간 00:00:01 | 입력 0 | 출력 0 | 초당 토큰 0.0") {
		t.Fatal(text)
	}
}

func TestRunSummaryReportedCostOrder(t *testing.T) {
	f := Formatter{Language: locale.English}
	s := runrecord.Statistics{ElapsedMillis: 1000, GenerationMillis: 1000, Usage: &modelapi.TokenUsage{Input: 20, Output: 5, Cost: new(0.001)}}
	if text := f.runSummary(s).Text; !strings.Contains(text, "INPUT 20 | OUTPUT 5 | TOTAL COST $0.001 | TPS 5.0") {
		t.Fatal(text)
	}
	f.Language = locale.Korean
	if text := f.runSummary(s).Text; !strings.Contains(text, "입력 20 | 출력 5 | 총 비용 $0.001 | 초당 토큰 5.0") {
		t.Fatal(text)
	}
}

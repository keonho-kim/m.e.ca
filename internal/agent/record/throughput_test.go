// Purpose: Verify weighted throughput ignores unmeasured samples and preserves real zero counts.
// Pattern: Pure measurement contract tests.
// Usage: go test ./internal/agent/record -run TestThroughput.
// Related: internal/agent/record/throughput.go, internal/agent/record/event_validation.go.
package record

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func TestThroughputWeightsCountsAndIntervals(t *testing.T) {
	total := MeasuredThroughput(Event{Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: 2 * time.Second})
	total = total.Add(MeasuredThroughput(Event{Usage: &modelapi.TokenUsage{Output: 30}, Generation: &Generation{MeasuredMillis: 1000}}))
	if rate := total.Rate(); rate == nil || *rate != 50.0/3 {
		t.Fatalf("expected 50 tokens / 3 seconds: %+v rate=%v", total, rate)
	}
	for _, event := range []Event{
		{Usage: &modelapi.TokenUsage{Output: 200}},
		{MeasuredDuration: time.Second},
		{Usage: &modelapi.TokenUsage{Output: -1}, MeasuredDuration: time.Second},
		{Usage: &modelapi.TokenUsage{Output: 20}, MeasuredDuration: -time.Second},
	} {
		if next := total.Add(MeasuredThroughput(event)); next != total {
			t.Fatalf("unmeasured sample affected throughput: %+v", event)
		}
	}
	zero := MeasuredThroughput(Event{Usage: &modelapi.TokenUsage{}, MeasuredDuration: time.Second})
	if rate := zero.Rate(); rate == nil || *rate != 0 {
		t.Fatal("actual reported zero was lost")
	}
	if rate := (Throughput{}).Rate(); rate != nil {
		t.Fatal("missing measurement was fabricated")
	}
}

func TestThroughputLivePrecisionAndNoDuplicatePersistence(t *testing.T) {
	event := Event{Kind: "usage", Usage: &modelapi.TokenUsage{Output: 1}, MeasuredDuration: 500 * time.Microsecond}
	if rate := MeasuredThroughput(event).Rate(); rate == nil || *rate != 2000 {
		t.Fatal("live measurement lost sub-millisecond precision")
	}
	if err := event.Check(); err == nil {
		t.Fatal("transient clock was accepted by persistence")
	}
	body, err := json.Marshal(event)
	if err != nil || strings.Contains(string(body), "MeasuredDuration") || strings.Contains(string(body), "500000") {
		t.Fatalf("transient duration was duplicated in storage: %s %v", body, err)
	}
	event.MeasuredDuration = 0
	if err := event.Check(); err != nil {
		t.Fatal("removing the transient sample did not leave a valid usage record:", err)
	}
}

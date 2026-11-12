// Purpose: Keep terminal history metadata independent from agent and checkpoint narrative.
// Pattern: Public projection contract regression tests.
// Usage: go test ./internal/serverapi -run TestHistoryCompletion.
// Related: internal/serverapi/projection.go, api/meca/v1/state.proto.
package serverapi

import (
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/agent/work"
	"github.com/keonho-kim/meca/internal/storage/history"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestHistoryCompletionExcludesNarrative(t *testing.T) {
	for _, status := range []work.Status{work.Completed, work.Failed, work.Interrupted, work.Denied, work.Exhausted} {
		t.Run(string(status), func(t *testing.T) {
			const narrative = "AGENT_NARRATIVE_MUST_STAY_OUT_OF_STATISTICS"
			value := historyItem(history.Record{Sequence: 7, Kind: "event", Event: record.Event{
				Kind: "run_checkpoint", Text: narrative, Thinking: narrative,
				Checkpoint: &record.Checkpoint{Phase: "finished", End: &work.End{Status: status, Reason: narrative}},
				Statistics: &record.Statistics{ElapsedMillis: 9000, GenerationMillis: 3000},
			}})
			body, err := protojson.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if value.GetEvent() != nil || value.GetMessage() != nil || strings.Contains(string(body), narrative) {
				t.Fatal("completion metadata carries message content", string(body))
			}
			completion := value.GetCompletion()
			if completion.GetStatus().String() != strings.ToUpper(string(status)) ||
				completion.GetStatistics().GetElapsedMillis() != 9000 || completion.GetStatistics().GetGenerationMillis() != 3000 {
				t.Fatal("terminal status or measurements were lost", string(body))
			}
		})
	}
}

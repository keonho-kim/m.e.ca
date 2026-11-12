// Purpose: Keep narrative, team contributions, fixed notices and measurements in exclusive wire bodies.
// Pattern: Public presentation contract regressions.
// Usage: go test ./internal/serverapi -run TestPresentation.
// Related: internal/serverapi/projection.go, api/meca/v1/events.proto, api/meca/v1/state.proto.
package serverapi

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/commands/server"
	"github.com/keonho-kim/meca/internal/storage/history"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestPresentationBodiesAreExclusive(t *testing.T) {
	for _, tc := range []struct{ kind, body string }{
		{"progress", "message"}, {"team_progress", "message"},
		{"team_message", "message"}, {"team_discussion", "message"},
		{"thinking", "message"}, {"assistant_delta", "message"},
		{"team_work", "work"}, {"retry", "notice"},
		{"compaction_started", "notice"}, {"compaction_finished", "notice"},
		{"team_notice", ""}, {"request_started", ""}, {"future_event", ""},
	} {
		t.Run(tc.kind, func(t *testing.T) {
			wire := event(server.Event{Record: record.Event{
				Kind: tc.kind, Agent: "Hana", Text: "NARRATIVE", Thinking: "REASONING",
				Team: &record.TeamNote{Work: &record.TeamWork{Revision: 1, Paths: []string{"api/"}}},
			}})
			body, err := protojson.Marshal(wire)
			if err != nil {
				t.Fatal(err)
			}
			var value map[string]json.RawMessage
			if err := json.Unmarshal(body, &value); err != nil {
				t.Fatal(err)
			}
			if value["text"] != nil || value["thinking"] != nil || value["team"] != nil {
				t.Fatalf("unclassified narrative escaped: %s", body)
			}
			count := 0
			for _, name := range []string{"message", "work", "notice", "tool", "completion"} {
				if value[name] != nil {
					count++
					if name != tc.body {
						t.Fatalf("wrong body: %s", body)
					}
				}
			}
			if (tc.body == "" && count != 0) || (tc.body != "" && count != 1) {
				t.Fatalf("missing or mixed content: %s", body)
			}
			if tc.body != "message" && tc.body != "work" && (strings.Contains(string(body), "NARRATIVE") || strings.Contains(string(body), "REASONING")) {
				t.Fatalf("free text reached control or notice: %s", body)
			}
		})
	}
}

func TestPresentationHistoryHasOneEntry(t *testing.T) {
	item := historyItem(history.Record{Sequence: 7, Kind: "event", Event: record.Event{
		Kind: "team_work", Agent: "Hana", Text: "My contribution",
		Team: &record.TeamNote{Work: &record.TeamWork{Revision: 2}},
	}})
	body, err := protojson.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]json.RawMessage
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	if value["kind"] != nil || value["text"] != nil || value["role"] != nil {
		t.Fatalf("parallel history models: %s", body)
	}
	if value["event"] == nil || value["completion"] != nil || value["message"] != nil {
		t.Fatalf("invalid history union: %s", body)
	}
}

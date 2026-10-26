// Purpose: Verify actual charge totals restore from usage events without estimating missing costs.
// Pattern: Repository contract tests.
// Usage: go test ./internal/storage/sqlite -run TestCost.
// Related: internal/storage/sqlite/usage.go, internal/agent/model/usage.go.
package sqlite

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

func TestCostRestoresReportedTotals(t *testing.T) {
	for _, tc := range []struct {
		name  string
		costs []string
		want  *float64
	}{
		{"unreported", []string{"null"}, nil},
		{"free", []string{"0"}, new(0.0)},
		{"paid and unreported", []string{"0.002", "null", "0.003"}, new(0.005)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store, err := newTestStore(t)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := store.Close(); err != nil {
					t.Error(err)
				}
			})
			for _, cost := range tc.costs {
				var event runrecord.Event
				if err := json.Unmarshal([]byte(`{"kind":"usage","usage":{"input":20,"output":5,"cost":`+cost+`}}`), &event); err != nil {
					t.Fatal(err)
				}
				if err := store.RecordEvent(context.Background(), store.Session().ID, event); err != nil {
					t.Fatal(err)
				}
			}
			total, err := store.TokenUsage(context.Background(), store.Session().ID)
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(total)
			var got struct {
				Cost *float64 `json:"cost"`
			}
			if err != nil || json.Unmarshal(body, &got) != nil {
				t.Fatal("invalid usage serialization")
			}
			if (got.Cost == nil) != (tc.want == nil) || got.Cost != nil && math.Abs(*got.Cost-*tc.want) > 1e-12 {
				t.Fatalf("cost was lost or estimated: %s", body)
			}
		})
	}
}

func TestCostRejectsInvalidStoredAmounts(t *testing.T) {
	for _, cost := range []string{`"0.01"`, `true`, `-1`, `1e999`} {
		t.Run(cost, func(t *testing.T) {
			store, err := newTestStore(t)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() {
				if err := store.Close(); err != nil {
					t.Error(err)
				}
			})
			ctx := context.Background()
			body := `{"kind":"usage","usage":{"input":20,"output":5,"cost":` + cost + `}}`
			_, err = store.db.ExecContext(ctx, "INSERT INTO tool_events(session_id,kind,body,raw_bytes,created_at) VALUES(?,'usage',?,?,?)", store.Session().ID, body, len(body), "2026-10-25T00:00:00Z")
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.TokenUsage(ctx, store.Session().ID); err == nil {
				t.Fatal("invalid stored charge was accepted")
			}
			var retained string
			if err := store.db.QueryRowContext(ctx, "SELECT body FROM tool_events WHERE session_id=? ORDER BY sequence DESC LIMIT 1", store.Session().ID).Scan(&retained); err != nil || retained != body {
				t.Fatalf("stored record changed: %s %v", retained, err)
			}
		})
	}
}

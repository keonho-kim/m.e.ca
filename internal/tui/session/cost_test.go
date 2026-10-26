// Purpose: Verify OpenRouter-only charges appear in order and snapshots are not counted twice.
// Pattern: Virtual terminal contract tests.
// Usage: go test ./internal/tui/session -run TestCost.
// Related: internal/tui/session/tokens.go, internal/tui/view/tokens.go.
package session

import (
	"encoding/json"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/tui/locale"
)

func TestCostHeaderSnapshotsAndProviderSwitch(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.selection.Connection.Kind = connection.OpenRouter
	a.screen.SetSize(140, 24)
	a.render()
	if line := screenLine(a.screen, 0, 140); !strings.Contains(line, "INPUT 0 · OUTPUT 0 · TOTAL COST — · TPS —") {
		t.Fatal(line)
	}
	update := func(kind, cost string) {
		t.Helper()
		var usage modelapi.TokenUsage
		if err := json.Unmarshal([]byte(`{"input":20,"output":5,"cost":`+cost+`}`), &usage); err != nil {
			t.Fatal(err)
		}
		a.receiveRuntime(&runrecord.Event{Kind: kind, Usage: &usage})
	}
	update("usage_progress", "0.001")
	update("usage_progress", "0.002")
	update("usage", "0.002")
	a.receiveRuntime(&runrecord.Event{Kind: "generation_started"})
	update("usage", "0.003")
	for _, page := range []string{"chat", "model", "language", "files"} {
		a.page = page
		a.render()
		if line := screenLine(a.screen, 0, 140); !strings.Contains(line, "INPUT 40 · OUTPUT 10 · TOTAL COST $0.005 · TPS —") {
			t.Fatalf("%s: %s", page, line)
		}
	}
	a.language = locale.Korean
	a.render()
	if line := screenLine(a.screen, 0, 140); !strings.Contains(line, "입력 40 · 출력 10 · 총 비용 $0.005 · 초당 토큰 —") {
		t.Fatal(line)
	}
	a.selection.Connection.Kind = connection.VLLM
	a.render()
	if line := screenLine(a.screen, 0, 140); strings.Contains(line, "비용") || strings.Contains(line, "$0.005") {
		t.Fatal("non-OpenRouter connection displays a cost: " + line)
	}
}

func TestCostFreeCallAndNarrowCompression(t *testing.T) {
	a := fixture(t)
	a.language = locale.English
	a.selection.Connection.Kind = connection.OpenRouter
	a.screen.SetSize(140, 24)
	var usage modelapi.TokenUsage
	if err := json.Unmarshal([]byte(`{"input":20,"output":5,"cost":0}`), &usage); err != nil {
		t.Fatal(err)
	}
	a.receiveRuntime(&runrecord.Event{Kind: "usage", Usage: &usage})
	a.render()
	if line := screenLine(a.screen, 0, 140); !strings.Contains(line, "TOTAL COST $0 · TPS —") {
		t.Fatal(line)
	}
	a.screen.SetSize(32, 12)
	a.startCompression()
	a.render()
	if line := screenLine(a.screen, 0, 32); !strings.Contains(line, "Compress Memory") {
		t.Fatal("cost hid the compression indicator: " + line)
	}
}

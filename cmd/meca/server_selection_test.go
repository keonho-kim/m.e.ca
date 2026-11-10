// Purpose: Verify server reads and execution preserve persisted selections and validate defaults.
// Pattern: Deterministic selection-precedence regression test.
// Usage: go test ./cmd/meca -run TestServerSelection.
// Related: cmd/meca/server_selection.go, cmd/meca/server_resources.go, cmd/meca/server_session.go.
package main

import (
	"testing"

	"github.com/keonho-kim/meca/internal/config"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func TestServerSelection(t *testing.T) {
	defaults := &config.SavedModel{Provider: connection.VLLM, Endpoint: "http://127.0.0.1:8000", Name: "default-model"}
	explicit, err := defaults.Selection()
	if err != nil {
		t.Fatal(err)
	}
	explicit.Model = "conversation-model"
	for _, test := range []struct {
		name     string
		selected connection.Selection
		fallback *config.SavedModel
		model    string
		invalid  bool
	}{
		{name: "default", fallback: defaults, model: "default-model"},
		{name: "persisted", selected: explicit, fallback: defaults, model: "conversation-model"},
		{name: "persisted before invalid default", selected: explicit, fallback: &config.SavedModel{}, model: "conversation-model"},
		{name: "unconfigured"},
		{name: "invalid default", fallback: &config.SavedModel{}, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := serverSelection(test.selected, test.fallback)
			if (err != nil) != test.invalid || got.Model != test.model {
				t.Fatal(got, err)
			}
		})
	}
}

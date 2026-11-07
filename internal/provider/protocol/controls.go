// Purpose: Translate explicit reasoning choices through the selected serving API contract.
// Pattern: Provider adapter.
// Usage: Generate resolves ControlsFor before any HTTP call; absent choices preserve server defaults.
// Related: internal/provider/connection/reasoning_options.go, internal/provider/protocol/native_controls.go.
package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/chat"
)

type Controls struct {
	OmitThinkingSummary bool
	Reasoning           *chat.Reasoning
	Think               any
	Template            map[string]any
	ReasoningEffort     string
}

func ControlsFor(kind connection.ProviderKind, info *connection.ModelInfo, request modelapi.Request) (Controls, error) {
	if info != nil && info.ID != request.ModelID {
		return Controls{}, fmt.Errorf("request model differs from prepared model")
	}
	options := connection.ReasoningOptions(kind, info)
	controls, err := nativeControls(kind, &options, request)
	if info != nil && info.ThoughtSummary != nil && !*info.ThoughtSummary {
		controls.OmitThinkingSummary = true
	}
	return controls, err
}

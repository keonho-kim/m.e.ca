// Purpose: Translate explicit reasoning choices through the selected serving API contract.
// Pattern: Provider adapter.
// Usage: Generate resolves ControlsFor before any HTTP call; absent choices preserve server defaults.
// Related: internal/provider/connection/reasoning_options.go, internal/provider/protocol/native_controls.go.
package protocol

import (
	"fmt"
	modelapi "github.com/keonho-kim/v-code/internal/agent/model"
	"github.com/keonho-kim/v-code/internal/provider/connection"
)

type Controls struct {
	Think           any
	Template        map[string]any
	ReasoningEffort string
}

func ControlsFor(kind connection.ProviderKind, info *connection.ModelInfo, request modelapi.Request) (Controls, error) {
	if info != nil && info.ID != request.ModelID {
		return Controls{}, fmt.Errorf("request model differs from prepared model")
	}
	options := connection.ReasoningOptions(kind, info)
	return nativeControls(kind, &options, request)
}

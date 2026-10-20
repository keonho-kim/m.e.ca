// Purpose: Determine advertised native or reviewed profile Thinking toggle eligibility.
// Pattern: Pure Function.
// Usage: Setup supplies discovered metadata to select controls without generating a response.
// Related: internal/provider/connection/reasoning.go, internal/provider/profile/effort.go.
package diagnostic

import (
	"fmt"
	modelconn "github.com/keonho-kim/v-code/internal/provider/connection"
	"github.com/keonho-kim/v-code/internal/provider/profile"
)

func ThinkingControl(kind modelconn.ProviderKind, model, profileID string, info *modelconn.ModelInfo) (bool, error) {
	if kind == modelconn.Ollama || kind == modelconn.LMStudio {
		if info == nil {
			return false, fmt.Errorf("native model metadata is required")
		}
		r := info.Reasoning
		return r != nil && r.Enabled() && r.Off, nil
	}
	selected, err := profile.Resolve(model, profileID)
	if err != nil {
		return false, err
	}
	return len(selected.ThinkingKeys(kind)) > 0, nil
}

// Purpose: Validate model-bound connection settings independently of storage and model names.
// Pattern: Simple Module.
// Usage: Storage adapters validate selections on both write and read.
// Related: internal/provider/connection/models.go, internal/provider/connection/reasoning_options.go.
package selection

import (
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

func Validate(selected connection.Selection) error {
	parsed, err := connection.New(selected.Connection.Kind, selected.Connection.Endpoint)
	if err != nil {
		return err
	}
	if parsed.ID != selected.Connection.ID || strings.TrimSpace(selected.Model) == "" || len(selected.Model) > 512 || strings.ContainsAny(selected.Model, "\r\n\x00") {
		return faults.Errorf(diagcatalog.SelectionInvalidModelSelection)
	}
	if selected.Identity != nil {
		if err := selected.Identity.Validate(); err != nil {
			return err
		}
	}
	if info := selected.Details; info != nil {
		if info.ID != selected.Model {
			return faults.Errorf(diagcatalog.SelectionModelMetadataBindingMismatch)
		}
		if err := info.Check(); err != nil {
			return err
		}
	}
	options := connection.ReasoningOptions(selected.Connection.Kind, selected.Details)
	if selected.ThinkingSupported && (!options.Enabled() || !options.Off) {
		return faults.Errorf(diagcatalog.SelectionInvalidONOFFRequestCapability)
	}
	if selected.Thinking != nil && (*selected.Thinking && !options.Enabled() || !*selected.Thinking && !options.Off) {
		return faults.Errorf(diagcatalog.SelectionUnsupportedThinkingChoice)
	}
	if selected.Effort() != "" && selected.Thinking == nil {
		return faults.Errorf(diagcatalog.SelectionEffortRequiresAnExplicitThinkingSetting)
	}
	return options.CheckEffort(selected.Effort())
}

// Purpose: Confirm final native arguments without changing their streamed values.
// Pattern: Bounded argument finalization.
// Usage: Responses decoding finalizes arguments in the shared accumulator.
// Related: internal/provider/protocol/native/calls.go, internal/provider/protocol/native/argument_identity.go.
package native

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (d *Draft) Final(value string) error {
	if d.Arguments.Len() == 0 {
		return d.Append(value)
	}
	if d.Arguments.String() != value && !SameArguments(d.Arguments.String(), value) {
		return &modelapi.ToolCallError{Detail: diagcatalog.Text(diagcatalog.NativeFinalFunctionArgumentsDifferFromStreamedArguments)}
	}
	return nil
}

// Purpose: Validate native candidates without side effects before scheduling a batch.
// Pattern: Pure input boundary.
// Usage: Single and multi-call dispatch share this validation contract.
// Related: internal/agent/run/action.go, internal/agent/run/batch.go.
package run

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type validatedCall struct {
	call       toolapi.Call
	definition toolapi.Definition
}

func (s *execution) validateCall(original toolapi.Call) (validatedCall, error) {
	call := original
	var definition *toolapi.Definition
	for i := range s.definitions {
		if s.definitions[i].Name == call.Name {
			definition = &s.definitions[i]
			break
		}
	}
	if err := call.CheckIdentity(); err != nil {
		return validatedCall{}, err
	}
	if definition == nil {
		return validatedCall{}, faults.Errorf(diagcatalog.RunUnknownToolSelectADeclaredNativeFunction)
	}
	if len(call.Arguments) > s.limits.MaxArgumentBytes {
		return validatedCall{}, faults.Errorf(diagcatalog.RunArgumentsExceedTheConfiguredByteLimitNarrow, s.limits.MaxArgumentBytes)
	}
	_, err := toolapi.ValidateArguments(*definition, call.Arguments)
	if err != nil {
		hint := "Correct only the invalid argument; no operation was executed."
		if call.Name == "edit_file" {
			hint = "Use edits:[{old_text,new_text}] from completed reads of the unchanged original. old_text must match once; a read in this batch cannot supply source for its dependent edit."
		}
		return validatedCall{}, faults.Errorf(diagcatalog.RunArgumentValidationFailed, err, hint)
	}

	return validatedCall{call, *definition}, nil
}

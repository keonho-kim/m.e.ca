// Purpose: Validate native candidates without side effects before scheduling a batch.
// Pattern: Pure input boundary.
// Usage: Single and multi-call dispatch share this validation contract.
// Related: internal/agent/run/action.go, internal/agent/run/batch.go.
package run

import (
	"errors"

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
	var err error
	if validator, ok := s.runner.Tools.(toolapi.CallArgumentValidator); ok {
		err = validator.ValidateCallArguments(call, *definition)
	} else {
		_, err = toolapi.ValidateArguments(*definition, call.Arguments)
	}
	if err != nil {
		hint := "Correct the invalid argument and reuse the other arguments. The call did not execute."
		var invalid *toolapi.ArgumentError
		if call.Name == "edit_file" && errors.As(err, &invalid) && invalid.Name == "edits" {
			hint = "Use edits:[{old_text,new_text}] from completed reads of the unchanged original. old_text must match once; a read in this batch cannot supply source for its dependent edit."
		}
		return validatedCall{}, faults.Errorf(diagcatalog.RunArgumentValidationFailed, err, hint)
	}

	return validatedCall{call, *definition}, nil
}

func (s *execution) validateCalls(calls []toolapi.Call) ([]validatedCall, []error) {
	validated := make([]validatedCall, len(calls))
	var problems []error
	for i, call := range calls {
		value, err := s.validateCall(call)
		if err != nil {
			if problems == nil {
				problems = make([]error, len(calls))
			}
			problems[i] = err
		}
		validated[i] = value
	}
	if problems != nil {
		return nil, problems
	}
	for i := range validated {
		validated[i].call = s.identifyCall(validated[i].call)
	}
	return validated, nil
}

// Purpose: Admit model-facing code calls independently of native host-tool arguments.
// Pattern: Input boundary for the sole model execution tool.
// Usage: The decision loop validates the complete response before saving or running code.
// Related: internal/tools/codemode/catalog.go, internal/agent/run/codemode.go.
package run

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/codemode"
)

func (s *execution) validateCodeCalls(calls []toolapi.Call) ([]validatedCall, []error) {
	definition := s.codeDefinition
	validated := make([]validatedCall, len(calls))
	var problems []error
	for i, call := range calls {
		call.ParentExecutionID = ""
		err := call.CheckIdentity()
		if err == nil && call.Name != codemode.Name {
			err = faults.New(diag.InvalidArgument, diag.CodeOnlyTool)
		}
		if err == nil && len(call.Arguments) > s.limits.MaxArgumentBytes {
			err = faults.Errorf(diag.RunArgumentsExceedTheConfiguredByteLimitNarrow, s.limits.MaxArgumentBytes)
		}
		if err == nil {
			_, err = toolapi.ValidateArguments(definition, call.Arguments)
		}
		if err != nil {
			if problems == nil {
				problems = make([]error, len(calls))
			}
			problems[i] = err
		} else {
			validated[i] = validatedCall{s.identifyCall(call), definition}
		}
	}
	return validated, problems
}

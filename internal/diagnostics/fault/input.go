// Purpose: Preserve the offending input field with a centrally defined validation error.
// Pattern: Structured boundary error.
// Usage: Compatible request parsers attach field names without defining error messages.
// Related: internal/diagnostics/catalog/messages.go, internal/serverapi/openai/json.go.
package fault

import "github.com/keonho-kim/meca/internal/diagnostics/catalog"

type InputError struct {
	Param string
	Cause error
}

func Invalid(param string, message catalog.Message, args ...any) error {
	return &InputError{Param: param, Cause: New(catalog.InvalidArgument, message, args...)}
}
func (e *InputError) Error() string { return e.Cause.Error() }
func (e *InputError) Unwrap() error { return e.Cause }

func InputParam(param string, cause error) error { return &InputError{Param: param, Cause: cause} }

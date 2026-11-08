// Purpose: Construct product errors from centrally declared templates while preserving causes.
// Pattern: Structured error value.
// Usage: Product packages create errors here and adapters render registered public messages.
// Related: internal/diagnostics/catalog/messages.go, internal/diagnostics/fault/classify.go.
package fault

import (
	"errors"
	"fmt"
	"strings"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type Error struct {
	code    string
	message error
	public  string
}

func New(code string, message catalog.Message, args ...any) *Error {
	if len(args) == 0 {
		return &Error{code: code, message: errors.New(string(message)), public: string(message)}
	}
	var publicArgs []any
	for i, arg := range args {
		if cause, ok := arg.(error); ok {
			if publicArgs == nil {
				publicArgs = append([]any(nil), args...)
			}
			publicArgs[i] = catalog.Text(catalog.PublicMessage(Code(cause)))
		}
	}
	original := fmt.Errorf(string(message), args...)
	if publicArgs == nil {
		return &Error{code: code, message: original, public: original.Error()}
	}
	public := fmt.Sprintf(strings.ReplaceAll(string(message), "%w", "%v"), publicArgs...)
	return &Error{code: code, message: original, public: public}
}

func Errorf(message catalog.Message, args ...any) error {
	return New(catalog.MessageCode(message), message, args...)
}
func (e *Error) Error() string     { return e.message.Error() }
func (e *Error) ErrorCode() string { return e.code }
func (e *Error) Unwrap() error {
	if _, ok := e.message.(interface{ Unwrap() error }); ok {
		return e.message
	}
	if _, ok := e.message.(interface{ Unwrap() []error }); ok {
		return e.message
	}
	return nil
}

func PublicMessage(err error, redact func(string) string) string {
	if err == nil {
		return ""
	}
	if redact == nil {
		return catalog.Text(catalog.UnknownFailure)
	}
	code := Code(err)
	var known *Error
	if !errors.As(err, &known) || catalog.Priority(code) >= 100 {
		return catalog.Text(catalog.PublicMessage(code))
	}
	return redact(known.public)
}

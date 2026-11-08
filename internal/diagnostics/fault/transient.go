// Purpose: Mark an observed recoverable response failure without retrying arbitrary internal errors.
// Pattern: Explicit error classification marker.
// Usage: Provider response parsers mark incomplete streams and malformed metadata at their I/O boundary.
// Related: internal/provider/transport/retry.go, internal/provider/protocol/chunk.go.
package fault

import (
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type transientError struct{ cause error }

func (e *transientError) Error() string     { return e.cause.Error() }
func (e *transientError) Unwrap() error     { return e.cause }
func (e *transientError) ErrorCode() string { return diagcatalog.Unavailable }

func Transient(cause error) error {
	if cause == nil {
		return nil
	}
	return &transientError{cause}
}
func IsTransient(err error) bool { var value *transientError; return errors.As(err, &value) }

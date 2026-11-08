// Purpose: Convert unexpected boundary panics into fixed private failures without exposing panic values.
// Pattern: Owned-operation failure boundary.
// Usage: Request, tool, provider-worker and logging owners settle their own resources after Guard.
// Related: internal/agent/run/execute.go, internal/commands/server/execution.go, internal/diagnostics/writer.go.
package fault

import (
	"errors"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type PanicError struct{}

func (*PanicError) Error() string     { return diagcatalog.Text(diagcatalog.UnexpectedFailure) }
func (*PanicError) ErrorCode() string { return diagcatalog.Internal }

func Panic() error           { return &PanicError{} }
func IsPanic(err error) bool { var value *PanicError; return errors.As(err, &value) }

func Guard(operation func() error) (err error) {
	defer func() {
		if recover() != nil {
			err = Panic()
		}
	}()
	return operation()
}

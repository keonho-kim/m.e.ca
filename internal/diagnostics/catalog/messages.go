// Purpose: Define centrally owned error templates and safe presentation fallbacks.
// Pattern: Declarative diagnostic catalog.
// Usage: Fault constructors accept only catalog templates; declarations remain in this package.
// Related: internal/diagnostics/fault/error.go, internal/diagnostics/catalog/codes.go.
package catalog

import "fmt"

type Message string

const (
	WrappedCause         Message = "%w"
	UnknownFailure       Message = "operation failed; inspect the recorded task state before retrying"
	ObserverDisconnected Message = "observer disconnected; retrieve the task state before retrying"
	ObserverDeadline     Message = "observer deadline exceeded; retrieve the task state before retrying"
	DeliveryUnavailable  Message = "could not deliver the response; retrieve the recorded task result"
	CompletionUncertain  Message = "could not confirm persisted task completion; inspect recorded state before retrying"
)

func Text(message Message, args ...any) string {
	if len(args) == 0 {
		return string(message)
	}
	return fmt.Sprintf(string(message), args...)
}

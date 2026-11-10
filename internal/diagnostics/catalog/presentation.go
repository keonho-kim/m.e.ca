// Purpose: Own public failure fallbacks and protocol code translations.
// Pattern: Declarative presentation policy.
// Usage: Native adapters render the same classified cause.
// Related: internal/diagnostics/fault/classify.go, internal/diagnostics/catalog/codes.go.
package catalog

func NativeCode(code string) string {
	if code == OperationFailed {
		return Internal
	}
	return code
}

func PublicMessage(code string) Message {
	switch code {
	case StorageBusy:
		return StorageBusyMessage
	case StorageReadOnly:
		return StorageReadOnlyMessage
	case StorageFull:
		return StorageFullMessage
	case StorageCorrupt:
		return StorageCorruptMessage
	case Canceled:
		return ObserverDisconnected
	case DeadlineExceeded:
		return ObserverDeadline
	case DeliveryFailed:
		return DeliveryUnavailable
	case PersistenceFailed, CompletionNotSaved, IntegrityFailed, StorageIO:
		return CompletionUncertain
	default:
		return UnknownFailure
	}
}

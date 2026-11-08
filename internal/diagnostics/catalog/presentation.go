// Purpose: Own public failure fallbacks and protocol code translations.
// Pattern: Declarative presentation policy.
// Usage: Native and compatible adapters render the same classified cause.
// Related: internal/diagnostics/fault/classify.go, internal/diagnostics/catalog/codes.go.
package catalog

const (
	MethodNotAllowed    = "method_not_allowed"
	ModelNotFound       = "model_not_found"
	InvalidRequest      = "invalid_request"
	InvalidAPIKey       = "invalid_api_key"
	Forbidden           = "forbidden"
	InvalidParameter    = "invalid_parameter"
	ServerError         = "server_error"
	InvalidRequestError = "invalid_request_error"
	AuthenticationError = "authentication_error"
	RateLimitError      = "rate_limit_error"
)

func NativeCode(code string) string {
	if code == OperationFailed {
		return Internal
	}
	return code
}

func CompatibleCode(code string, parameter bool) string {
	if parameter && code == InvalidArgument {
		return InvalidParameter
	}
	if code == Canceled {
		return ObserverCanceled
	}
	if code == OperationFailed || code == Internal {
		return ServerError
	}
	return code
}

func CompatibleType(status int) string {
	if status >= 500 {
		return ServerError
	}
	if status == 401 {
		return AuthenticationError
	}
	if status == 429 {
		return RateLimitError
	}
	return InvalidRequestError
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
	case Canceled, ObserverCanceled:
		return ObserverDisconnected
	case DeadlineExceeded:
		return ObserverDeadline
	case DeliveryFailed:
		return DeliveryUnavailable
	case StreamRewrite:
		return StreamRewritten
	case StreamGap:
		return StreamBehind
	case PersistenceFailed, CompletionNotSaved, IntegrityFailed, StorageIO:
		return CompletionUncertain
	default:
		return UnknownFailure
	}
}

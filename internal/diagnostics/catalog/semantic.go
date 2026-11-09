// Purpose: Normalize registered tool and transport identities into shared failure classes.
// Pattern: Declarative error taxonomy.
// Usage: Central classification and persisted event validation reject arbitrary product codes.
// Related: internal/diagnostics/catalog/codes.go, internal/diagnostics/fault/tool.go.
package catalog

func Semantic(code string) string {
	switch code {
	case StorageBusy:
		return Unavailable
	case StorageReadOnly:
		return FailedPrecondition
	case StorageFull:
		return ResourceExhausted
	case StorageCorrupt:
		return IntegrityFailed
	case StorageIO:
		return PersistenceFailed
	case MethodNotAllowed, InvalidRequest:
		return InvalidArgument
	case InvalidAPIKey, Forbidden:
		return PermissionDenied
	case ModelNotFound:
		return NotFound
	case ServerError:
		return OperationFailed
	case CommandBlocked, AccessRejectedCode:
		return PermissionDenied
	case InvalidCommand, InvalidParameter:
		return InvalidArgument
	case StaleTarget, InstructionChanged, AmbiguousText, TextNotFound, TeamWorkConflict:
		return Conflict
	case Timeout:
		return DeadlineExceeded
	case Interrupted:
		return Canceled
	case ProcessFailed, ExecutionFailed, BatchNotExecuted, AdmissionFailed, LogWriteFailed:
		return OperationFailed
	case PathNotFound:
		return NotFound
	case TeamCommandPending, TeamContributionRequired, TeamFeedbackPending, ApprovalRequired, SessionClosed:
		return FailedPrecondition
	case OutputTruncated, DiagnosticRecordLimitCode:
		return ResourceExhausted
	case RotationDeferredCode:
		return Unavailable
	case HttpHandlerPanic, HttpServerError, TlsHandshakeFailed:
		return Internal
	default:
		return code
	}
}

func ValidCode(code string) bool { return code == "" || Priority(Semantic(code)) > 0 }

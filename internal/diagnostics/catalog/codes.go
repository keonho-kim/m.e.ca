// Purpose: Own product error identities and their shared semantic severity.
// Pattern: Declarative diagnostic catalog.
// Usage: Fault classification, durable records and protocol adapters use these identities.
// Related: internal/diagnostics/fault/classify.go, internal/diagnostics/catalog/messages.go.
package catalog

const (
	AccessRejectedCode       = "access_rejected"
	AmbiguousText            = "ambiguous_text"
	ApprovalRequired         = "approval_required"
	BatchNotExecuted         = "batch_not_executed"
	SessionClosed            = "session_closed"
	AdmissionFailed          = "admission_failed"
	HttpHandlerPanic         = "http_handler_panic"
	HttpServerError          = "http_server_error"
	PathNotFound             = "path_not_found"
	TeamCommandPending       = "team_command_pending"
	TeamContributionRequired = "team_contribution_required"
	TeamFeedbackPending      = "team_feedback_pending"
	TeamWorkConflict         = "team_work_conflict"
	TextNotFound             = "text_not_found"
	TlsHandshakeFailed       = "tls_handshake_failed"
	CommandBlocked           = "command_blocked"
	InvalidCommand           = "invalid_command"
	StaleTarget              = "file_changed"
	InstructionChanged       = "user_instruction_changed"
	OutputTruncated          = "output_truncated"
	Timeout                  = "timeout"
	Interrupted              = "interrupted"
	ProcessFailed            = "process_failed"
	ExecutionFailed          = "execution_failed"
	Internal                 = "internal"
	OperationFailed          = "operation_failed"
	InvalidArgument          = "invalid_argument"
	NotFound                 = "not_found"
	PermissionDenied         = "permission_denied"
	Conflict                 = "conflict"
	FailedPrecondition       = "failed_precondition"
	ResourceExhausted        = "resource_exhausted"
	Unavailable              = "unavailable"
	Canceled                 = "canceled"
	DeadlineExceeded         = "deadline_exceeded"
	CompletionNotSaved       = "completion_not_saved"
	PersistenceFailed        = "persistence_failed"
	IntegrityFailed          = "integrity_failed"
	ObserverCanceled         = "observer_cancelled"
	StreamRewrite            = "stream_rewrite"
	StreamGap                = "stream_gap"
	DeliveryFailed           = "delivery_failed"
)

func Priority(code string) int {
	code = Semantic(code)
	switch code {
	case IntegrityFailed, PersistenceFailed, CompletionNotSaved:
		return 100
	case ResourceExhausted:
		return 90
	case PermissionDenied, FailedPrecondition:
		return 80
	case Internal, OperationFailed, Unavailable, DeliveryFailed, StreamRewrite, StreamGap:
		return 70
	case InvalidArgument, Conflict:
		return 60
	case NotFound:
		return 50
	case DeadlineExceeded:
		return 30
	case Canceled, ObserverCanceled:
		return 20
	default:
		return 0
	}
}

func HTTPStatus(code string) int {
	code = Semantic(code)
	switch code {
	case InvalidArgument:
		return 400
	case PermissionDenied:
		return 403
	case NotFound:
		return 404
	case Conflict, FailedPrecondition, Canceled, ObserverCanceled:
		return 409
	case ResourceExhausted:
		return 429
	case Unavailable:
		return 503
	case DeadlineExceeded:
		return 504
	default:
		return 500
	}
}

// Purpose: Own the complete set of diagnostic events and their default levels.
// Pattern: Declarative event catalog.
// Usage: Product packages emit registered events through diagnostics.Log.
// Related: internal/diagnostics/reporter.go, internal/diagnostics/catalog/declarations_01.go.
package catalog

import "log/slog"

type Event string

const (
	RunStarted             Event = Event(RunRunStarted)
	RunFinished            Event = Event(RunRunFinished)
	RequestStarted         Event = Event(ServerRequestStarted)
	RequestFinished        Event = Event(ServerRequestFinished)
	CompletionSaveFailed   Event = Event(ServerRequestCompletionNotSaved)
	StorageCommitFailed    Event = Event(SqliteStorageCommitFailed)
	StorageCommitSlow      Event = Event(SqliteStorageCommitSlow)
	StorageCommitted       Event = Event(SqliteStorageCommitted)
	StorageWriterWait      Event = Event(SqliteStorageWriterWait)
	StorageCapacityLow     Event = "storage_capacity_low"
	ControlCompleted       Event = Event(ServerapiControlCompleted)
	StreamClosed           Event = Event(ServerapiStreamClosed)
	AccessRejected         Event = Event(ServerapiHttpRejected)
	ServerListening        Event = Event(MecaServerListening)
	RotationDeferred       Event = Event(CredentialsCredentialRotationDeferred)
	ApprovalWaiting        Event = Event(RunApprovalWaiting)
	ApprovalDecided        Event = Event(RunApprovalDecided)
	UIOperationFailed      Event = Event(SessionUiOperationFailed)
	BrowserLaunchFailed    Event = "browser_launch_failed"
	FolderSelectionFailed  Event = "folder_selection_failed"
	HTTPServerError        Event = "http_server_error"
	ResponseDeliveryFailed Event = "response_delivery_failed"
	DetailCacheBypassed    Event = "detail_cache_bypassed"
	ProcessFailedEvent     Event = "process_failed"
	ReporterMissing        Event = "diagnostics_not_configured"
	ReporterFailed         Event = "diagnostics_output_failed"
	ReporterSummary        Event = "diagnostics_summary"
)

func Level(event Event) (slog.Level, bool) {
	switch event {
	case StorageCommitFailed, CompletionSaveFailed, HTTPServerError, ProcessFailedEvent, ReporterFailed:
		return slog.LevelError, true
	case StorageCommitSlow, StorageWriterWait, StorageCapacityLow, AccessRejected, RotationDeferred, UIOperationFailed, BrowserLaunchFailed, FolderSelectionFailed, ResponseDeliveryFailed, ReporterMissing, DetailCacheBypassed:
		return slog.LevelWarn, true
	case RunStarted, RunFinished, RequestStarted, RequestFinished, ServerListening, ApprovalWaiting, ApprovalDecided, ReporterSummary:
		return slog.LevelInfo, true
	case StorageCommitted, ControlCompleted, StreamClosed:
		return slog.LevelDebug, true
	case "skill_loaded", "approved", "approval_required":
		return slog.LevelInfo, true
	case "tool_failed", "denied", "retry":
		return slog.LevelWarn, true
	case "generation_started", "generation_finished", "usage", "tool_started", "tool_succeeded", "tool_interrupted", "compaction_started", "compaction_finished":
		return slog.LevelDebug, true
	default:
		return slog.LevelDebug, false
	}
}

func RuntimeEvent(kind string) (Event, bool) {
	switch kind {
	case "skill_loaded", "approved", "approval_required", "tool_failed", "denied", "retry", "generation_started", "generation_finished", "usage", "tool_started", "tool_succeeded", "tool_interrupted", "compaction_started", "compaction_finished":
		return Event(kind), true
	default:
		return "", false
	}
}

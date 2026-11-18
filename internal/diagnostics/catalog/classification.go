// Purpose: Bind shared sentinel templates to stable product failure classes.
// Pattern: Declarative error classification.
// Usage: Fault constructors retain these semantics through wrapping and joining.
// Related: internal/diagnostics/catalog/messages.go, internal/diagnostics/fault/error.go.
package catalog

func MessageCode(message Message) string {
	switch message {
	case BrowserInput:
		return InvalidArgument
	case BrowserScope, BrowserImage, BrowserClosed, BrowserResult, BrowserStale:
		return FailedPrecondition
	case BrowserLimit:
		return ResourceExhausted
	case BrowserIntegrity, BrowserArchive:
		return IntegrityFailed
	case BrowserUnsupported, BrowserRuntime, BrowserLibraryMissing, BrowserProcessFailed:
		return Unavailable

	case SkillsInvalidSkillRequest, SessionInvalidOrExpiredSessionPageCursorRefresh:
		return InvalidArgument
	case SkillsSkillNotFound, ServerHistoryRecordWasNotFound, SessionSessionWasNotFound, ServerRequestWasNotFound, OsstoreOSCredentialWasNotFound:
		return NotFound
	case SkillsSkillReadLimitExceeded, HistoryStoredOriginalExceedsTheReadBudgetNarrow, AdmissionServerWorkExceedsItsAdmissionBudget, CapacityStorageCapacityLimitReachedExistingRecordsAnd, SessionSessionListSnapshotLimitReachedRefreshRetry:
		return ResourceExhausted
	case SkillsSkillRequiresAnExplicitUserMention, AgentrequestApprovalRequiredInNoninteractiveModeNoOperation, RunOperationDeniedByUserExecutionStopped:
		return FailedPrecondition
	case RunRuntimeObservationCouldNotBeCommittedOr:
		return PersistenceFailed
	case CredentialstoreCredentialAuthenticationFailedExistingDataWasPreserved, SessionSessionDoesNotBelongToThisDatabase:
		return IntegrityFailed
	case CredentialstoreOSEncryptionKeyUnavailablePreserveTheCredential:
		return Unavailable
	case RunModelCallBudgetExhausted:
		return ResourceExhausted
	case RunExecutionTimeBudgetExhaustedBeforeTheNext:
		return DeadlineExceeded
	default:
		return ""
	}
}

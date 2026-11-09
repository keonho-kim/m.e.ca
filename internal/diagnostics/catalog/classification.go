// Purpose: Bind shared sentinel templates to stable product failure classes.
// Pattern: Declarative error classification.
// Usage: Fault constructors retain these semantics through wrapping and joining.
// Related: internal/diagnostics/catalog/messages.go, internal/diagnostics/fault/error.go.
package catalog

func MessageCode(message Message) string {
	switch message {
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

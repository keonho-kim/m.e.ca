// Purpose: Own formatted error details and actionable tool recovery instructions.
// Pattern: Declarative error message catalog.
// Usage: Domain error values and central tool failures render these templates.
// Related: internal/diagnostics/fault/tool.go, internal/agent/model/model.go.
package catalog

const (
	AdmissionUncertain         Message = "request admission failed; no automatic replay is allowed"
	ResponseInterrupted        Message = "request was interrupted; its effects must not be replayed"
	ClosedSessionRecovery      Message = "The session has closed. Do not retry these calls. Its completed work and final report remain in the recorded conversation."
	ContextFailureRecovery     Message = "Restore the failed context or journal access before making another decision. Inspect saved outcomes and the latest user instructions before considering any remaining work."
	InterruptedBatchRecovery   Message = "Execution was interrupted or its budget ended. Inspect saved outcomes and any partial effects before resuming only work the user still requires."
	IndexedEditFailure         Message = "%s edits[%d]: %s"
	CommandCaptureUnavailable  Message = "command effects may have completed but its output could not be finalized; inspect saved state and partial effects before retrying"
	CommandCaptureFailed       Message = "command %s output could not be preserved: %s"
	UnexpectedFailure          Message = "an unexpected internal failure interrupted work; inspect recorded effects before restarting"
	ContextMemoryLimit         Message = "simultaneous model contexts exceed the process memory budget; inspect active work before retrying"
	DetailMaskChanged          Message = "protected credentials changed while preparing detail; retry the read"
	DetailMaskFactoryRequired  Message = "detail streaming requires a credential mask factory"
	DetailUnsupportedValue     Message = "unsupported public detail value"
	ProviderDetailOmitted      Message = "provider detail exceeded the diagnostic limit and was omitted"
	ToolFailureDescription     Message = "%s: %s; next: %s"
	ToolInspectState           Message = "Inspect current state and choose a narrower or different approach; do not blindly replay side effects."
	ToolInspectTimeout         Message = "Inspect partial output, process state and the remaining task budget before resuming; do not blindly replay side effects."
	ToolInspectInterruption    Message = "Inspect saved state before resuming; execution may have started."
	ToolInspectPath            Message = "List the parent directory and verify the literal path."
	InvalidToolCall            Message = "invalid tool call: %s"
	ModelOutputTruncated       Message = "model output truncated: %s"
	InvalidShellCommand        Message = "invalid shell command: %v"
	ImageInputUnavailableCause Message = "image input unavailable: %s"
	ImageInputUnconfirmed      Message = "image input has not been confirmed for the selected model"
	ModelContextUnavailable    Message = "model context limit exceeded; history was preserved"
	ModelContextExhausted      Message = "model context exhausted: input %d tokens, capacity %d; history was preserved"
	ProviderAttemptsFailed     Message = "provider failed after %d attempts: %s"
	ProviderGenerationFailure  Message = "provider generation error"
	ProviderHTTPFailure        Message = "HTTP %d (%s)"
	ProviderFailureCode        Message = "%s [%s]"
	ProviderFailureDetail      Message = "%s: %s"
)

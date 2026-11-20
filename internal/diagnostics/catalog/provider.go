// Purpose: Own diagnostics for provider stream progress, sequence errors and agent reply recovery.
// Pattern: Declarative diagnostic catalog.
// Usage: The transport, Responses decoder and execution loop use these diagnostic messages.
// Related: internal/provider/transport/content_watch.go.
package catalog

const ProviderNoContentProgress Message = "provider sent no generation progress for %s"

const (
	RunEmptyReply      Message = "model completed without an answer or native tool call"
	RunRetryEmptyReply Message = "Retrying after an empty model response"
)

const RunContinueAnswer Message = "Continuing the partial answer within the measured context budget"

const ResponsesInvalidSequence Message = "invalid Responses event sequence number"

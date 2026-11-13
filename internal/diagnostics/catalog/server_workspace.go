// Purpose: Own messages for session-scoped server control and workspace reads.
// Pattern: Central diagnostic catalog.
// Usage: Server application and wire adapters reject unsupported or ambiguous requests.
// Related: internal/commands/server/target.go, internal/serverapi/workspace.go.
package catalog

const (
	ServerExplicitSession           Message = "provide a valid session_id for this operation"
	ServerWorkspaceUnavailable      Message = "workspace access is unavailable"
	ServerPageInvalid               Message = "invalid page, offset or source reference"
	ServerSearchQueryInvalid        Message = "search query must contain 1 to 256 UTF-8 bytes"
	ServerModelDiscoveryUnavailable Message = "model discovery is unavailable"
	ServerImageMissing              Message = "the requested history image was not found"
	ServerQueueBytes                Message = "queued input exceeds its text memory budget"
	ServerReadRevision              Message = "this read revision is stale; restart from the first page"
	ServerOriginalReadBusy          Message = "the original read slot is occupied; retry after the current read completes"
	ServerBrowseBusy                Message = "all metadata query slots are occupied"
	ServerTaskSessionMismatch       Message = "the task does not belong to the requested session"
)

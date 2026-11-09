// Purpose: Own diagnostics for bounded concurrent tasks and multimodal server input.
// Pattern: Declarative diagnostic catalog.
// Usage: Server admission, attachment adapters and clipboard input use these messages.
// Related: internal/commands/server/service.go, internal/agent/media/image.go.
package catalog

const (
	ServerConcurrencyHelp   Message = "maximum simultaneous server tasks (positive integer)"
	ServerConcurrencyRange  Message = "server concurrency must be a positive integer"
	ServerTaskCapacity      Message = "all server task slots are occupied"
	ServerSessionTaskActive Message = "this session already owns an active task"
	ServerInvalidTaskFilter Message = "task_id must contain at most 128 bytes"
	ServerApprovalDecision  Message = "provide one JSON boolean named allow"
	DetailSpoolBudget       Message = "detail cache exceeds its temporary-storage budget; paging continues without a spool"
)

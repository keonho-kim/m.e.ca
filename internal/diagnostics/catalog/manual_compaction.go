// Purpose: Define user-visible outcomes for explicitly requested conversation compaction.
// Pattern: Diagnostic catalog entries.
// Usage: The session runtime and command adapters report manual compaction outcomes.
// Related: internal/agent/run/manual_compaction.go, internal/commands/server/tasks.go.
package catalog

const (
	ManualCompactionEmpty    Message = "No new conversation history to compact."
	ManualCompactionInput    Message = "!compact does not accept images or file references."
	ManualCompactionSteering Message = "Wait for conversation compaction to finish before sending a direction change."
	ManualCompactionDone     Message = "Conversation context compacted."
)

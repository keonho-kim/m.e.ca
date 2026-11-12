// Purpose: Own terminal recall retention independently of persisted conversation and request limits.
// Pattern: Shared consumer-specific resource policy.
// Usage: SQLite restoration and the terminal input-history state use the same bounds.
// Related: internal/storage/sqlite/input_history.go, internal/tui/session/input_history.go.
package history

const MaxRecentInputs = 200
const RecentInputBytes = 16 << 20

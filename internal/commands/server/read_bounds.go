// Purpose: Bound native history previews and explicitly requested original detail restoration.
// Pattern: Consumer-owned read contract limits.
// Usage: History and OpenDetail pass these distinct budgets to persistence adapters.
// Related: internal/commands/server/snapshot.go, internal/commands/server/detail.go.
package server

const MaxReadBytes = 32 << 20
const MaxOriginalBytes = 512 << 20

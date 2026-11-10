// Purpose: Own the accepted UTF-8 prompt size independently of browser draft retention.
// Pattern: Domain input contract.
// Usage: Task admission and native capabilities share MaxPromptBytes.
// Related: internal/commands/server/tasks.go, internal/commands/server/attachment_request.go, internal/serverapi/workspace.go.
package server

const MaxPromptBytes = 16 << 20

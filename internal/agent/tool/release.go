// Purpose: Return optional preparation resources on every execution or rejection path.
// Pattern: Narrow resource ownership contract.
// Usage: The run owner releases prepared single and batch operations; ordinary value-only tools need no hook.
// Related: internal/agent/tool/execution.go, internal/agent/run/prepare_execution.go.
package tool

// Release returns resources retained by preparation. Implementations are idempotent.
func Release(operation Execution) {
	if owned, ok := operation.(interface{ Release() }); ok {
		owned.Release()
	}
}

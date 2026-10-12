// Purpose: Define bounded command output returned by the OS process adapter.
// Pattern: Simple Module (adapter result).
// Usage: The local tool executor translates Output into its persisted tool-result contract.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/command.go.
package process

type Output struct {
	Stdout, Stderr string
	ExitCode       int
	Truncated      bool
}

// Purpose: Describe the observed process state carried by a tool result.
// Pattern: Immutable tool outcome value.
// Usage: Command tools report process status through Result and EffectFacts.
// Related: internal/agent/tool/execution.go, internal/tools/local/command_result.go.
package tool

type ProcessOutcome struct {
	CommandID  string
	Command    string
	Diagnostic string
	Truncated  bool
	Status     string
	ExitCode   int
}

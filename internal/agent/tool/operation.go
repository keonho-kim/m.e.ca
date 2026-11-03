// Purpose: Describe observed tool execution facts independently of compaction.
// Pattern: Domain record.
// Usage: Runtime and compaction preserve Operation records.
// Related: internal/agent/tool/execution.go, internal/agent/run/execution_result.go.
package tool

type Operation struct {
	AffectedPaths                                []string `json:"-"`
	Revision                                     int
	Target                                       string
	Versions                                     map[string]string
	Started                                      bool
	Failure                                      *Failure
	Process                                      *ProcessOutcome
	RunID, CallID, Name, Status, Command, Result string

	Changes []FileChange
}

// Purpose: Update one command observation and invalidate versions after possible shell changes.
// Pattern: In-place state projection.
// Usage: Facts.Observe delegates command-specific bookkeeping here.
// Related: internal/agent/tool/facts_observation.go, internal/agent/tool/facts.go.
package tool

func (f *Facts) observeCommand(op Operation) {
	status := "not_started"
	var exit *int
	if op.Started {
		status = "unknown"
		for i := range f.Files {
			f.Files[i].Stale = op.Versions[f.Files[i].Path] == ""
		}
	}
	if op.Process != nil {
		status = op.Process.Status
		if status != "running" {
			code := op.Process.ExitCode
			exit = &code
		}
		if op.Process.Command != "" {
			op.Command = op.Process.Command
		}
	}
	command, clipped := reportExcerpt(op.Command)
	fact := CommandFact{CallID: op.CallID, Command: command, Status: status, ExitCode: exit, Excerpt: clipped, ObservedRun: op.RunID, Revision: op.Revision}
	if op.Process != nil {
		fact.CommandID = op.Process.CommandID
	}
	for i, previous := range f.Commands {
		if fact.CommandID != "" && previous.CommandID == fact.CommandID {
			f.Commands[i] = fact
			return
		}
	}
	if command != "" {
		f.Commands = append(f.Commands, fact)
	}
}

// Purpose: Reuse small command labels while preserving vault masking revisions and acknowledgement cleanup.
// Pattern: Lifecycle-bound immutable metadata projection.
// Usage: Command activity and final collection share each owned command's label cache.
// Related: internal/tools/local/commands.go, internal/tools/local/command_live.go.
package local

func (e *Executor) SetRedactionVersion(version func() int) { e.redactionVersion = version }
func (e *Executor) commandVersion() int {
	if e.redactionVersion != nil {
		return e.redactionVersion()
	}
	return -1
}

func (e *Executor) commandLabel(id, command string) string {
	if e.redactionVersion == nil {
		return e.redact(command)
	}
	e.liveOutputs.mu.Lock()
	c := e.liveOutputs.values[id]
	if c == nil {
		if _, err := e.commands.Done(id); err != nil {
			e.liveOutputs.mu.Unlock()
			return e.redact(command)
		}
		if e.liveOutputs.values == nil {
			e.liveOutputs.values = make(map[string]*liveCommandCapture)
		}
		c = &liveCommandCapture{}
		e.liveOutputs.values[id] = c
	}
	e.liveOutputs.mu.Unlock()
	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	version := e.redactionVersion()
	if !c.commandCached || c.commandVersion != version {
		label := e.redact(command)
		c.command, c.commandCached = "", false
		if len(label) <= 256<<10 {
			c.command, c.commandCached, c.commandVersion = label, true, version
		}
		return label
	}
	return c.command
}

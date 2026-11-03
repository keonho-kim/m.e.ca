// Purpose: Retain known shell mutation targets only until their terminal observation is delivered.
// Pattern: Executor-owned immutable command metadata.
// Usage: Local command results register paths; team composition attaches and consumes them on events.
// Related: internal/tools/team/local.go, internal/commands/teamrequest/peer.go.
package team

func (e *Executor) trackCommandPaths(id string, paths []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.commandPaths == nil {
		e.commandPaths = make(map[string][]string)
	}
	e.commandPaths[id] = paths
}

// CommandTargets returns immutable paths, independent of any output capture.
// Terminal delivery releases the registry entry after the original is durable.
func (e *Executor) CommandTargets(id string, terminal bool) []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	paths := e.commandPaths[id]
	if terminal {
		delete(e.commandPaths, id)
		if len(e.commandPaths) == 0 {
			e.commandPaths = nil
		}
	}
	return paths
}

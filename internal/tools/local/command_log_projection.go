// Purpose: Replace retained logs even when a new credential mask makes the visible capture small.
// Pattern: Executor-owned full log projection.
// Usage: Live resets and final captures share the same stable artifact identity.
// Related: internal/tools/local/command_incremental.go, internal/tools/local/command_live.go.
package local

func (e *Executor) projectCaptureFile(id, stream, text string) (string, error) {
	path, err := e.commandPath(id, "."+stream+".log")
	if err != nil {
		return "", err
	}
	e.retained.mu.Lock()
	exists := false
	if entry := e.retained.entries[id]; entry != nil {
		_, exists = entry.files[path]
	}
	e.retained.mu.Unlock()
	if len(text) <= outputExcerptBytes && !exists {
		return "", nil
	}
	if err := e.writeArtifact(id, path, []byte(text)); err != nil {
		return "", err
	}
	return path, nil
}

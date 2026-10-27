// Purpose: Describe owned command status and bounded live output snapshots.
// Pattern: Immutable process projection.
// Usage: Registry wait/list results are translated by the local tool adapter.
// Related: internal/tools/local/process/registry.go, internal/tools/local/process/capture.go.
package process

import "time"

type State struct {
	ID               string "json:\"command_id\""
	CallID           string "json:\"-\""
	Command          string "json:\"command\""
	Directory        string "json:\"directory\""
	Status           string "json:\"status\""
	ElapsedMillis    int64  "json:\"elapsed_millis\""
	LastOutputMillis *int64 "json:\"last_output_millis\""
	ExitCode         int    "json:\"-\""
	Stdout, Stderr   string "json:\"-\""
	Error            string "json:\"-\""
}

func (j *job) snapshot(output bool) State {
	j.mu.Lock()
	state, ended := j.state, j.finished
	j.mu.Unlock()
	select {
	case <-j.done:
	default:
		state.Status = "running"
		state.ExitCode = -1
	}
	now := time.Now()
	if ended.IsZero() {
		ended = now
	}
	state.ElapsedMillis = ended.Sub(j.started).Milliseconds()
	var out, err time.Time
	out = j.stdout.lastOutput()
	err = j.stderr.lastOutput()
	if output {
		state.Stdout = j.stdout.String()
		state.Stderr = j.stderr.String()
	}
	if err.After(out) {
		out = err
	}
	if !out.IsZero() {
		age := max(0, now.Sub(out).Milliseconds())
		state.LastOutputMillis = &age
	}
	return state
}

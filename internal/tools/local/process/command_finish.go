// Purpose: Archive one completed command, release captures and publish its terminal state.
// Pattern: Joined command lifecycle.
// Usage: The command start path launches exactly one finish worker.
// Related: internal/tools/local/process/command_start.go, internal/tools/local/process/capture.go.
package process

import (
	"errors"
	"time"
)

func (r *Registry) finish(id string, j *job, pending startedCommand, save func(string, string, Output, error) error) {
	runErr := pending.wait()
	ended := time.Now()
	j.mu.Lock()
	j.finished = ended
	j.state.Status = "finalizing"
	j.mu.Unlock()
	release := r.ProjectionBudget().Acquire(j.stdout.retainedBytes() + j.stderr.retainedBytes())
	defer release()
	output, runErr := pending.output(runErr)
	var saveErr error
	if save != nil {
		saveErr = save(id, j.state.Command, output, runErr)
	}
	saveErr = errors.Join(saveErr, j.stdout.release(), j.stderr.release())
	j.mu.Lock()
	j.finished = ended
	j.state.Status, j.state.ExitCode = output.Status, output.ExitCode
	if saveErr != nil {
		j.state.Error = saveErr.Error()
	}
	j.mu.Unlock()
}

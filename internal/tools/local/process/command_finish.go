// Purpose: Archive one completed command, release captures and publish its terminal state.
// Pattern: Joined command lifecycle.
// Usage: The command start path launches exactly one finish worker.
// Related: internal/tools/local/process/command_start.go, internal/tools/local/process/capture.go.
package process

import (
	"errors"
	"time"
)

func (r *Registry) finish(id string, j *job, pending startedCommand, save func(string, Output, error) error) {
	defer func() {
		close(j.done)
		r.notify()
	}()
	defer j.cancel()
	runErr := pending.wait()
	r.captureMu.Lock()
	output, runErr := pending.output(runErr)
	var saveErr error
	if save != nil {
		saveErr = save(id, output, runErr)
	}
	saveErr = errors.Join(saveErr, j.stdout.release(), j.stderr.release())
	r.captureMu.Unlock()
	j.mu.Lock()
	j.finished = time.Now()
	j.state.Status, j.state.ExitCode = output.Status, output.ExitCode
	if saveErr != nil {
		j.state.Error = saveErr.Error()
	}
	j.mu.Unlock()
}

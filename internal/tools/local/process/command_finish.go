// Purpose: Archive one completed command, release captures and publish its terminal state.
// Pattern: Joined command lifecycle.
// Usage: The command start path launches exactly one finish worker.
// Related: internal/tools/local/process/command_start.go, internal/tools/local/process/capture.go.
package process

import (
	"context"
	"errors"
	"time"

	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func (r *Registry) finish(ctx context.Context, id string, j *job, pending startedCommand, save func(string, string, Output, error) error) {
	runErr := pending.wait()
	ended := time.Now()
	j.mu.Lock()
	j.finished = ended
	j.state.Status = "finalizing"
	j.mu.Unlock()
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()
	release, admissionErr := r.ProjectionBudget().Acquire(flush, j.stdout.retainedBytes()+j.stderr.retainedBytes())
	if admissionErr != nil {
		failure := errors.Join(runErr, faults.New(diagcatalog.PersistenceFailed, diagcatalog.CommandCaptureUnavailable), admissionErr)
		output := Output{Started: true, Status: diagcatalog.ExecutionFailed, ExitCode: -1, Truncated: true}
		if save != nil {
			failure = errors.Join(failure, faults.Guard(func() error { return save(id, j.state.Command, output, failure) }))
		}
		failure = errors.Join(failure, j.stdout.release(), j.stderr.release())
		j.mu.Lock()
		j.state.Status = diagcatalog.ExecutionFailed
		j.state.Error = failure.Error()
		j.mu.Unlock()
		return
	}
	defer release()
	output, runErr := pending.output(runErr)
	var saveErr error
	if save != nil {
		saveErr = faults.Guard(func() error { return save(id, j.state.Command, output, runErr) })
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

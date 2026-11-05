// Purpose: Read live output timing and release completed capture storage safely.
// Pattern: Synchronized capture projection.
// Usage: Registry snapshots share collectors with the process pipe writers.
// Related: internal/tools/local/process/output.go, internal/tools/local/process/registry.go.
package process

import (
	"bytes"
	"time"
)

func (output *cappedOutput) lastOutput() time.Time {
	output.timingMu.Lock()
	defer output.timingMu.Unlock()
	return output.lastWrite
}

func (output *cappedOutput) release() error {
	output.mu.Lock()
	defer output.mu.Unlock()
	output.head = bytes.Buffer{}
	output.tail = nil
	output.tailStart = 0
	output.truncated = false
	if output.spill != nil {
		err := output.spill.close()
		output.spill = nil
		return err
	}
	return nil
}

func (output *cappedOutput) isTruncated() bool {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.truncated || output.spill != nil && output.spill.truncated
}

func (output *cappedOutput) captureError() error {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.failure
}

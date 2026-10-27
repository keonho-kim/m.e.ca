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
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.lastWrite
}

func (output *cappedOutput) release() {
	output.mu.Lock()
	defer output.mu.Unlock()
	output.head = bytes.Buffer{}
	output.tail = nil
}

func (output *cappedOutput) isTruncated() bool {
	output.mu.Lock()
	defer output.mu.Unlock()
	return output.truncated
}

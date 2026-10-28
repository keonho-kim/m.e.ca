// Purpose: Capture bounded command output while retaining its beginning and final diagnostics.
// Pattern: Bounded head-and-tail collector.
// Usage: The native waiter and registry expose captures with explicit truncation and exit status.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/command.go.
package process

import (
	"bytes"
	"strings"
	"sync"
	"time"
)

type Output struct {
	StdoutPath, StderrPath                   string
	StdoutCapturedBytes, StderrCapturedBytes int
	Started                                  bool
	Status                                   string
	Stdout, Stderr                           string
	ExitCode                                 int
	Truncated                                bool
}

type cappedOutput struct {
	mu        sync.Mutex
	lastWrite time.Time
	head      bytes.Buffer
	tail      []byte
	tailStart int
	truncated bool
	revision  uint64
}

const captureHalf = 8 * 1024 * 1024

func (output *cappedOutput) Write(content []byte) (int, error) {
	output.mu.Lock()
	defer output.mu.Unlock()
	if len(content) > 0 {
		output.lastWrite = time.Now()
		output.revision++
	}
	count := len(content)
	headCount := min(len(content), captureHalf-output.head.Len())
	if _, err := output.head.Write(content[:headCount]); err != nil {
		return 0, err
	}
	content = content[headCount:]
	if len(content) >= captureHalf {
		output.truncated = output.truncated || len(content) > captureHalf || len(output.tail) > 0
		output.tail = append(output.tail[:0], content[len(content)-captureHalf:]...)
		output.tailStart = 0
	} else {
		space := captureHalf - len(output.tail)
		if space > 0 {
			n := min(space, len(content))
			output.tail = append(output.tail, content[:n]...)
			content = content[n:]
		}
		if len(content) > 0 {
			output.truncated = true
			n := copy(output.tail[output.tailStart:], content)
			copy(output.tail, content[n:])
			output.tailStart = (output.tailStart + len(content)) % captureHalf
		}
	}
	return count, nil
}

func (output *cappedOutput) String() string {
	return output.snapshot(0).Text
}

type Capture struct {
	Revision uint64
	Text     string
}

func (output *cappedOutput) snapshot(after uint64) Capture {
	output.mu.Lock()
	defer output.mu.Unlock()
	if output.revision == after {
		return Capture{Revision: after}
	}
	var text strings.Builder
	text.Grow(output.head.Len() + len(output.tail) + 32)
	text.Write(output.head.Bytes())
	if output.truncated {
		text.WriteString("\n[... output omitted ...]\n")
	}
	text.Write(output.tail[output.tailStart:])
	text.Write(output.tail[:output.tailStart])
	return Capture{Revision: output.revision, Text: text.String()}
}

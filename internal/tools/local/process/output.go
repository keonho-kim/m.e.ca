// Purpose: Capture bounded command output while retaining its beginning and final diagnostics.
// Pattern: Bounded head-and-tail collector.
// Usage: Run translates these captures into a result with explicit truncation and exit status.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/command.go.
package process

import (
	"bytes"
	"strings"
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
	head      bytes.Buffer
	tail      []byte
	tailStart int
	truncated bool
}

const captureHalf = 8 * 1024 * 1024

func (output *cappedOutput) Write(content []byte) (int, error) {
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
	var text strings.Builder
	text.Grow(output.head.Len() + len(output.tail) + 32)
	text.Write(output.head.Bytes())
	if output.truncated {
		text.WriteString("\n[... output omitted ...]\n")
	}
	text.Write(output.tail[output.tailStart:])
	text.Write(output.tail[:output.tailStart])
	return text.String()
}

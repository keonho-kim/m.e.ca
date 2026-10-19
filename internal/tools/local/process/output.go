// Purpose: Capture bounded command output while retaining its beginning and final diagnostics.
// Pattern: Bounded head-and-tail collector.
// Usage: Run translates these captures into a result with explicit truncation and exit status.
// Related: internal/tools/local/process/run_unix.go, internal/tools/local/command.go.
package process

import "bytes"

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
	} else {
		if excess := len(output.tail) + len(content) - captureHalf; excess > 0 {
			copy(output.tail, output.tail[excess:])
			output.tail = output.tail[:len(output.tail)-excess]
			output.truncated = true
		}
		output.tail = append(output.tail, content...)
	}
	return count, nil
}

func (output *cappedOutput) String() string {
	if output.truncated {
		return output.head.String() + "\n[... output omitted ...]\n" + string(output.tail)
	}
	return output.head.String() + string(output.tail)
}

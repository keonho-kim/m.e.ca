// Purpose: Reconstruct a command original from its existing logs only at the durable commit boundary.
// Pattern: Deferred projection with request-shared memory admission.
// Usage: The runtime requests Capture before recording and acknowledges it after persistence.
// Related: internal/tools/local/command_archive.go, internal/tools/local/command_live.go.
package local

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/keonho-kim/meca/internal/tools/local/process"
)

// CommandCapture does not create another full-output file beside the readable logs.
func (e *Executor) CommandCapture(id string) (json.RawMessage, error) {
	if _, err := e.commands.Done(id); err != nil {
		return nil, nil
	}
	result, err := e.readCommandResult(id)
	if err != nil {
		return nil, err
	}
	var output process.Output
	if err = json.Unmarshal(result.Content, &output); err != nil {
		return nil, err
	}
	if output.StdoutPath == "" && output.StderrPath == "" {
		return nil, nil
	}
	size := int64(output.StdoutCapturedBytes + output.StderrCapturedBytes)
	release := e.commands.ProjectionBudget().Acquire(size)
	c := e.liveOutputs.command(id)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.captureRelease != nil {
		release()
		return nil, fmt.Errorf("command output is already borrowed for commit; acknowledge this command before collecting it again")
	}
	for i, path := range []string{output.StdoutPath, output.StderrPath} {
		if path == "" {
			continue
		}
		text, readErr := readCaptureText(path)
		if readErr != nil {
			release()
			return nil, readErr
		}
		if i == 0 {
			output.Stdout = text
		} else {
			output.Stderr = text
		}
	}
	output.StdoutPath, output.StderrPath = "", ""
	body, err := json.Marshal(output)
	if err != nil {
		release()
		return nil, err
	}
	c.captureRelease = release
	return body, nil
}

func readCaptureText(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	var text strings.Builder
	text.Grow(int(info.Size()))
	_, err = io.Copy(&text, file)
	return text.String(), err
}

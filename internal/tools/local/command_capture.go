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
	"syscall"

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
		expected := output.StdoutCapturedBytes
		if i == 1 {
			expected = output.StderrCapturedBytes
		}
		text, readErr := readCaptureText(path, expected)
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

func readCaptureText(path string, expected int) (string, error) {
	if expected < 0 || expected > maxArtifactBytes {
		return "", fmt.Errorf("invalid captured log size")
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(expected) {
		return "", fmt.Errorf("captured log changed before its durable record")
	}
	var text strings.Builder
	text.Grow(int(info.Size()))
	_, err = io.Copy(&text, io.LimitReader(file, int64(expected)+1))
	if err == nil && text.Len() != expected {
		err = fmt.Errorf("captured log changed while reading")
	}
	return text.String(), err
}

// Purpose: Reconstruct a command original from its existing logs only at the durable commit boundary.
// Pattern: Deferred projection with request-shared memory admission.
// Usage: The runtime requests Capture before recording and acknowledges it after persistence.
// Related: internal/tools/local/command_archive.go, internal/tools/local/command_live.go.
package local

import (
	"context"
	"encoding/json"
	"errors"

	"io"
	"os"
	"strings"
	"syscall"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/tools/local/process"

	// CommandCapture does not create another full-output file beside the readable logs.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (e *Executor) CommandCapture(ctx context.Context, id string) (json.RawMessage, error) {
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
	if output.StdoutCapturedBytes < 0 || output.StderrCapturedBytes < 0 || output.StdoutCapturedBytes > maxArtifactBytes || output.StderrCapturedBytes > maxArtifactBytes {
		return nil, faults.Errorf(diagcatalog.LocalInvalidCapturedLogSize)
	}
	size := int64(output.StdoutCapturedBytes) + int64(output.StderrCapturedBytes)
	release, err := e.commands.ProjectionBudget().Acquire(ctx, size)
	if err != nil {
		return nil, err
	}
	c := e.liveOutputs.command(id)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.captureRelease != nil {
		release()
		return nil, faults.Errorf(diagcatalog.LocalCommandOutputIsAlreadyBorrowedForCommit)
	}
	for i, path := range []string{output.StdoutPath, output.StderrPath} {
		if path == "" {
			continue
		}
		expected := output.StdoutCapturedBytes
		if i == 1 {
			expected = output.StderrCapturedBytes
		}
		text, readErr := readCaptureText(ctx, path, expected)
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

func readCaptureText(ctx context.Context, path string, expected int) (result string, err error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if expected < 0 || expected > maxArtifactBytes {
		return "", faults.Errorf(diagcatalog.LocalInvalidCapturedLogSize)
	}
	file, err := os.OpenFile(path, os.O_RDONLY|syscall.O_NONBLOCK|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(expected) {
		return "", faults.Errorf(diagcatalog.LocalCapturedLogChangedBeforeItsDurableRecord)
	}
	var text strings.Builder
	text.Grow(int(info.Size()))
	buffer := make([]byte, 64<<10)
	for text.Len() < expected {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := file.Read(buffer[:min(len(buffer), expected-text.Len())])
		text.Write(buffer[:n])
		if err != nil && !errors.Is(err, io.EOF) {
			return "", err
		}
		if n == 0 || err != nil && text.Len() != expected {
			return "", faults.Errorf(diagcatalog.LocalCapturedLogChangedWhileReading)
		}
	}
	after, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if !os.SameFile(info, after) || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return "", faults.Errorf(diagcatalog.LocalCapturedLogChangedWhileReading)
	}
	return text.String(), ctx.Err()
}

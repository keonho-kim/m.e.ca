// Purpose: Convert completed command output into sanitized excerpts and typed outcomes.
// Pattern: Command result projection.
// Usage: The executor-owned command archive formats each terminal result once.
// Related: internal/tools/local/command_archive.go, internal/tools/local/output.go.
package local

import (
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
	"strings"
	"unicode/utf8"
)

func (e *Executor) encodeCommand(id string, output process.Output, runErr error, capture *liveCommandCapture) (toolapi.Result, error) {
	output.CommandID = id
	var archiveErr, projectionErr error
	texts := [2]string{output.Stdout, output.Stderr}
	for i, stream := range []string{"stdout", "stderr"} {
		previous := capture.streams[i]
		revision := output.Revisions[i]
		if revision == 0 {
			revision = previous.revision + 1
		}
		if revision != previous.revision || e.commandVersion() < 0 || previous.version != e.commandVersion() {
			previous.revision = 0
			previous, archiveErr = e.projectLiveCapture(id, stream, previous, process.Capture{Revision: revision, Text: texts[i]})
			projectionErr = errors.Join(projectionErr, archiveErr)
		}
		capture.streams[i] = previous
	}
	output.Stdout, output.StdoutPath = capture.streams[0].excerpt, capture.streams[0].path
	output.Stderr, output.StderrPath = capture.streams[1].excerpt, capture.streams[1].path
	output.StdoutCapturedBytes, output.StderrCapturedBytes = capture.streams[0].bytes, capture.streams[1].bytes
	runErr = errors.Join(runErr, projectionErr)
	result, err := resultfmt.Encode(output, fmt.Sprintf("Process %s; exit code %d", output.Status, output.ExitCode))
	if err != nil {
		return result, errors.Join(runErr, err)
	}
	result.Started = output.Started
	result.Process = &toolapi.ProcessOutcome{Status: output.Status, ExitCode: output.ExitCode, Truncated: output.Truncated}
	result.Failed = runErr != nil || output.ExitCode != 0 || output.Status != "exited"
	if result.Failed {
		result.Process.Diagnostic = commandDiagnostic(output.Stdout, output.Stderr)
	}
	if runErr != nil {
		result.Failure = toolapi.FailureOf(runErr)
	} else if result.Failed {
		result.Failure = &toolapi.Failure{Code: toolapi.ProcessFailed, Detail: result.Summary, Next: "Interpret stdout, stderr and the exit code for this command. grep/rg exit 1 means no matches. If this is the requested observation, report it; otherwise correct the cause and retry only the needed work."}
	}
	return result, projectionErr
}

func commandDiagnostic(stdout, stderr string) string {
	text := strings.TrimSpace(stderr)
	if text == "" {
		text = strings.TrimSpace(stdout)
	}
	if len(text) <= 16*1024*1024 {
		return text
	}
	start := len(text) - 16*1024*1024
	for !utf8.RuneStart(text[start]) {
		start++
	}
	return "[diagnostic tail; see original captured output]\n" + text[start:]
}

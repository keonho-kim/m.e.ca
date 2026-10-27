// Purpose: Convert completed command output into sanitized excerpts and typed outcomes.
// Pattern: Command result projection.
// Usage: The executor-owned command archive formats each terminal result once.
// Related: internal/tools/local/command_archive.go, internal/tools/local/output.go.
package local

import (
	"encoding/json"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"github.com/keonho-kim/meca/internal/tools/local/textformat"
	resultfmt "github.com/keonho-kim/meca/internal/tools/result"
	"strings"
	"unicode/utf8"
)

func (e *Executor) encodeCommand(output process.Output, runErr error) (toolapi.Result, error) {
	output.Stdout = textformat.RedactSource(output.Stdout, e.redact)
	output.Stderr = textformat.RedactSource(output.Stderr, e.redact)
	originalOutput := output
	output.StdoutCapturedBytes, output.StderrCapturedBytes = len(output.Stdout), len(output.Stderr)
	var archiveErr error
	output.Stdout, output.StdoutPath, archiveErr = e.commandOutput(output.Stdout, "stdout")
	runErr = errors.Join(runErr, archiveErr)
	output.Stderr, output.StderrPath, archiveErr = e.commandOutput(output.Stderr, "stderr")
	runErr = errors.Join(runErr, archiveErr)
	result, err := resultfmt.Encode(output, fmt.Sprintf("Process %s; exit code %d", output.Status, output.ExitCode))
	if err != nil {
		return result, err
	}
	if output.StdoutPath != "" || output.StderrPath != "" {
		result.Capture, err = json.Marshal(originalOutput)
		if err != nil {
			return result, errors.Join(runErr, err)
		}
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
	return result, runErr
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

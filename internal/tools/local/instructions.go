// Purpose: Read bounded project-root AGENTS.md through the confined file boundary.
// Pattern: Filesystem Adapter.
// Usage: Called once per user task; its sanitized snapshot is reused for all loop iterations.
// Related: internal/tools/local/instructions_test.go, internal/agent/prompt/prompt.go.
package local

import (
	"context"
	"errors"

	"io"
	"os"
	"syscall"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/policy"
)

func (e *Executor) ProjectInstructions(ctx context.Context) (text string, err error) {
	if err = ctx.Err(); err != nil {
		return "", err
	}
	path, err := e.resolve(policy.Directory, "AGENTS.md")
	if err != nil {
		return "", err
	}
	file, err := path.root.OpenFile(path.relative, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", faults.Errorf(diagcatalog.LocalProjectAGENTSMdMustBeARegular)
	}
	if info.Size() > prompt.MaxInstructionsBytes {
		return "", faults.Errorf(diagcatalog.LocalProjectAGENTSMdIsBytesMaximumIs, info.Size(), prompt.MaxInstructionsBytes)
	}
	body, err := io.ReadAll(io.LimitReader(file, prompt.MaxInstructionsBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > prompt.MaxInstructionsBytes {
		return "", faults.Errorf(diagcatalog.LocalProjectAGENTSMdGrewBeyondTheByte, prompt.MaxInstructionsBytes)
	}
	if _, err = prompt.Compose(prompt.Defaults().System, string(body)); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return e.redact(string(body)), nil
}

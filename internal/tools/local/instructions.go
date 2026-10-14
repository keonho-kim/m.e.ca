// Purpose: Read bounded project-root AGENTS.md through the confined file boundary.
// Pattern: Filesystem Adapter.
// Usage: Called once per user task; its sanitized snapshot is reused for all loop iterations.
// Related: internal/tools/local/instructions_test.go, internal/agent/prompt/prompt.go.
package local

import (
	"context"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/prompt"
	"github.com/keonho-kim/v-code/internal/policy"
	"io"
	"os"
	"syscall"
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
		return "", fmt.Errorf("project AGENTS.md must be a regular file")
	}
	if info.Size() > prompt.MaxInstructionsBytes {
		return "", fmt.Errorf("project AGENTS.md is %d bytes; maximum is %d bytes (1 MiB)", info.Size(), prompt.MaxInstructionsBytes)
	}
	body, err := io.ReadAll(io.LimitReader(file, prompt.MaxInstructionsBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > prompt.MaxInstructionsBytes {
		return "", fmt.Errorf("project AGENTS.md grew beyond the %d-byte limit (1 MiB) while reading", prompt.MaxInstructionsBytes)
	}
	if _, err = prompt.Compose(prompt.Defaults().System, string(body), false); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	return e.redact(string(body)), nil
}

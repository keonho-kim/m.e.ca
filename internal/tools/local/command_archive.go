// Purpose: Store sanitized completed command results outside the active in-memory registry.
// Pattern: Executor-owned ephemeral result archive.
// Usage: A command waiter archives once; tools and the runtime load results until session cleanup.
// Related: internal/tools/local/commands.go, internal/tools/local/executor.go.
package local

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/tools/local/process"
)

func (e *Executor) commandPath(id, suffix string) (string, error) {
	if !strings.HasPrefix(id, "cmd_") || len(id) > 128 || len(id) <= 4 || strings.ContainsFunc(id[4:], func(r rune) bool { return r > 127 || !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		return "", fmt.Errorf("invalid command_id; use an ID returned by this session")
	}
	return filepath.Join(e.artifacts, id+suffix), nil
}

func (e *Executor) archiveCommand(id string, output process.Output, runErr error) error {
	result, err := e.encodeCommand(output, runErr)
	if err != nil && result.Content == nil {
		return err
	}
	result.Process.CommandID = id
	for _, state := range e.commands.List() {
		if state.ID == id {
			result.Process.Command = e.redact(state.Command)
			break
		}
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(result.Content, &fields); err != nil {
		return err
	}
	fields["command_id"], _ = json.Marshal(id)
	result.Content, err = json.Marshal(fields)
	if err != nil {
		return err
	}
	path, pathErr := e.commandPath(id, ".json")
	if pathErr != nil {
		return pathErr
	}
	if len(result.Capture) > 0 {
		if err := os.WriteFile(path+".capture", result.Capture, 0600); err != nil {
			return err
		}
		result.Capture = nil
	}
	body, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return marshalErr
	}
	// Only the owning waiter writes; done is closed after this file is complete.
	return os.WriteFile(path, body, 0600)
}

func (e *Executor) readCommandResult(id string, capture bool) (toolapi.Result, error) {
	var result toolapi.Result
	path, err := e.commandPath(id, ".json")
	if err != nil {
		return result, err
	}
	body, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return result, fmt.Errorf("unknown command_id %q in this session", id)
	}
	if err != nil {
		return result, err
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return result, err
	}
	if capture {
		result.Capture, err = os.ReadFile(path + ".capture")
		if errors.Is(err, os.ErrNotExist) {
			err = nil
		}
	}
	return result, err
}

func (e *Executor) AcknowledgeCommand(id string) error {
	path, err := e.commandPath(id, ".json.capture")
	if err != nil {
		return err
	}
	if err = os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	e.commands.Acknowledge(id)
	e.liveOutputs.forget(id)
	return nil
}

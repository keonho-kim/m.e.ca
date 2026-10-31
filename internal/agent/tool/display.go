// Purpose: Select small presentation arguments without copying hidden file bodies.
// Pattern: Read-only projection of an already validated native call.
// Usage: SQLite stores this projection and terminal formatters consume it.
// Related: internal/tui/transcript/tool_arguments.go, internal/storage/sqlite/events.go.
package tool

import (
	"encoding/json"
	"errors"
	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
)

type DisplayArguments struct {
	Path        string   `json:"path,omitempty"`
	Command     string   `json:"command,omitempty"`
	CommandID   string   `json:"command_id,omitempty"`
	Paths       []string `json:"paths,omitempty"`
	WaitSeconds *int     `json:"wait_seconds,omitempty"`
}

// DisplayArgs does not authorize or validate execution. The selected values
// are copied so short labels cannot keep a large argument buffer alive.
func DisplayArgs(raw json.RawMessage) (DisplayArguments, error) {
	var args DisplayArguments
	node, err := sonic.GetWithOptions(raw, ast.SearchOptions{CopyReturn: false})
	if err != nil {
		return args, err
	}
	for _, field := range []struct {
		name   string
		target any
	}{{"path", &args.Path}, {"command", &args.Command}, {"command_id", &args.CommandID}, {"paths", &args.Paths}, {"wait_seconds", &args.WaitSeconds}} {
		value := node.Get(field.name)
		if err := value.Check(); errors.Is(err, ast.ErrNotExist) {
			continue
		} else if err != nil {
			return args, err
		}
		text, err := value.Raw()
		if err != nil {
			return args, err
		}
		if err = json.Unmarshal([]byte(text), field.target); err != nil {
			return args, err
		}
	}
	return args, nil
}

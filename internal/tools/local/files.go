// Purpose: Execute bounded file reads and approved atomic file mutations.
// Pattern: Filesystem Adapter.
// Usage: Invoked by prepared operations after policy and version rechecks.
// Related: internal/tools/local/executor.go, internal/tools/local/paths.go, internal/tools/local/files_test.go.

package local

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"unicode/utf8"

	"github.com/keonho-kim/v-code/internal/agent"
	"github.com/keonho-kim/v-code/internal/policy"
)

type Entry struct {
	Name      string `json:"name"`
	Directory bool   `json:"directory"`
}

func (executor *Executor) Directory(ctx context.Context, settings policy.Settings, path string, offset, limit int) ([]Entry, bool, error) {
	resolved, err := executor.resolve(settings.Scope, path)
	if err != nil {
		return nil, false, err
	}
	if _, err := policy.Decide(settings, false, policy.Read); err != nil {
		return nil, false, err
	}
	entries, more, err := readDirectory(ctx, resolved, offset, limit)
	if err != nil {
		return nil, false, err
	}
	visible := entries[:0]
	for _, entry := range entries {
		target, resolveErr := executor.resolve(settings.Scope, filepath.Join(resolved.absolute, entry.Name))
		if resolveErr == nil {
			info, statErr := target.root.Stat(target.relative)
			if statErr == nil {
				entry.Directory = info.IsDir()
				visible = append(visible, entry)
			}
		}
	}
	return visible, more, nil
}

func readDirectory(ctx context.Context, path target, offset, limit int) (entries []Entry, more bool, err error) {
	if offset < 0 || offset > 100000 || limit < 1 || limit > 1000 {
		return nil, false, fmt.Errorf("invalid directory page")
	}
	file, err := path.root.Open(path.relative)
	if err != nil {
		return nil, false, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()

	for {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		batch, readErr := file.ReadDir(1000)
		for _, entry := range batch {
			entries = append(entries, Entry{Name: entry.Name(), Directory: entry.IsDir()})
		}
		if len(entries) > 25000 {
			return nil, false, fmt.Errorf("directory exceeds 25000 entries")
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, false, readErr
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Directory != entries[j].Directory {
			return entries[i].Directory
		}
		return entries[i].Name < entries[j].Name
	})
	if offset >= len(entries) {
		return nil, false, nil
	}
	more = len(entries) > offset+limit
	entries = entries[offset:min(len(entries), offset+limit)]
	return entries, more, nil
}

func (op *operation) run(ctx context.Context) (agent.ToolResult, error) {
	switch op.spec.name {
	case "read_file":
		return op.readFile()
	case "list_directory":
		limit := op.input.Limit
		if limit == 0 {
			limit = 200
		}
		entries, more, err := op.executor.Directory(ctx, op.settings, op.source.absolute, op.input.Offset, limit)
		if err != nil {
			return agent.ToolResult{}, err
		}
		return encodeResult(struct {
			Entries    []Entry `json:"entries"`
			More       bool    `json:"more"`
			NextOffset int     `json:"next_offset"`
		}{entries, more, op.input.Offset + limit}, fmt.Sprintf("%d entries", len(entries)))
	case "search_files", "grep":
		return op.search(ctx)
	case "write_file", "edit_file":
		if err := op.writeFile(); err != nil {
			return agent.ToolResult{}, err
		}
	case "delete_file":
		if err := op.source.root.Remove(op.source.relative); err != nil {
			return agent.ToolResult{}, err
		}
	case "move_file":
		current, err := op.executor.resolve(op.settings.Scope, op.input.Destination)
		if err != nil {
			return agent.ToolResult{}, err
		}
		version, err := snapshot(current)
		if err != nil {
			return agent.ToolResult{}, err
		}
		if current.absolute != op.destination.absolute || !sameVersion(op.destinationVersion, version) {
			return agent.ToolResult{}, fmt.Errorf("destination changed after preparation")
		}
		if err := op.source.root.Rename(op.source.relative, op.destination.relative); err != nil {
			return agent.ToolResult{}, err
		}
	case "execute_command":
		return op.command(ctx)
	default:
		return agent.ToolResult{}, fmt.Errorf("unknown prepared tool")
	}
	result, err := encodeResult(struct {
		Path    string `json:"path"`
		Changed bool   `json:"changed"`
	}{op.source.absolute, true}, "Completed")
	if err != nil {
		return result, err
	}
	change := agent.FileChange{Action: op.spec.name, Path: op.source.absolute}
	if op.spec.name == "move_file" {
		change.Destination = op.destination.absolute
	}
	result.Changes = []agent.FileChange{change}
	return result, nil
}

func (op *operation) readFile() (result agent.ToolResult, err error) {
	file, err := op.source.root.Open(op.source.relative)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return result, err
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("only regular files can be read")
	}
	limit := op.input.Limit
	if limit == 0 {
		limit = 64 * 1024
	}
	if _, err := file.Seek(int64(op.input.Offset), io.SeekStart); err != nil {
		return result, err
	}
	content, err := io.ReadAll(io.LimitReader(file, int64(limit)))
	if err != nil {
		return result, err
	}
	if !utf8.Valid(content) {
		return result, fmt.Errorf("file range is not UTF-8 text")
	}
	return encodeResult(struct {
		Path, Content string
		Truncated     bool
	}{op.source.absolute, op.executor.redact(string(content)), info.Size() > int64(op.input.Offset+len(content))}, fmt.Sprintf("%d bytes read", len(content)))
}

func (op *operation) writeFile() (err error) {
	parent := filepath.Dir(op.source.relative)
	if err := op.source.root.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temporary := filepath.Join(parent, ".v-write-"+rand.Text())
	mode := os.FileMode(0o644)
	if op.original.exists {
		mode = op.original.info.Mode().Perm()
	}
	file, err := op.source.root.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := op.source.root.Remove(temporary); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	_, writeErr := file.Write(op.replacement)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	if err := errors.Join(writeErr, file.Close()); err != nil {
		return err
	}
	return op.source.root.Rename(temporary, op.source.relative)
}

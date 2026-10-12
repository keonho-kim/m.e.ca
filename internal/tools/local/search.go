// Purpose: Search scoped filenames and UTF-8 lines with bounded traversal and output.
// Pattern: Filesystem Adapter.
// Usage: Called for search_files and grep operations.
// Related: internal/tools/local/files.go, internal/tools/local/executor.go.

package local

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent"
)

type match struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

func (op *operation) search(ctx context.Context) (agent.ToolResult, error) {
	var expression *regexp.Regexp
	var err error
	if op.spec.name == "grep" {
		expression, err = regexp.Compile(op.input.Pattern)
	} else {
		_, err = filepath.Match(op.input.Pattern, "probe")
	}
	if err != nil {
		return agent.ToolResult{}, err
	}
	var matches []match
	visited, truncated := 0, false
	stop := errors.New("search limit reached")
	err = fs.WalkDir(op.source.root.FS(), op.source.relative, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		visited++
		if visited > 25000 || len(matches) >= 1000 {
			truncated = true
			return stop
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		absolute := filepath.Join(op.source.root.Name(), path)
		if absolute == op.executor.private || strings.HasPrefix(absolute, op.executor.private+string(filepath.Separator)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if op.spec.name == "search_files" {
			found, err := filepath.Match(op.input.Pattern, entry.Name())
			if err != nil {
				return err
			}
			if found {
				matches = append(matches, match{Path: absolute})
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
			return nil
		}
		file, err := op.source.root.Open(path)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 256*1024)
		line := 0
		for scanner.Scan() {
			line++
			if expression.MatchString(scanner.Text()) {
				matches = append(matches, match{Path: absolute, Line: line, Text: op.executor.redact(scanner.Text())})
			}
			if len(matches) >= 1000 {
				truncated = true
				break
			}
		}
		return errors.Join(scanner.Err(), file.Close())
	})
	if err != nil && !errors.Is(err, stop) {
		return agent.ToolResult{}, err
	}
	return encodeResult(struct {
		Matches   []match `json:"matches"`
		Truncated bool    `json:"truncated"`
	}{matches, truncated}, fmt.Sprintf("%d matches", len(matches)))
}

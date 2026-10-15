// Purpose: Execute bounded Plan filename and text queries using read-only filesystem APIs.
// Pattern: Filesystem read adapter.
// Usage: Local Plan command execution supplies a parsed query and resolved root.
// Related: internal/policy/shell/query.go, internal/tools/local/query.go.
package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/keonho-kim/v-code/internal/policy/shell"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
)

type Request struct {
	Root          *os.Root
	Path, Private string
	Query         shell.Query
	Redact        func(string) string
}
type Result struct {
	Text      string
	Truncated bool
	Matches   int
}

func (r Request) Execute(ctx context.Context) (Result, error) {
	q := r.Query
	pattern := q.Pattern
	if q.Content && q.Fixed {
		pattern = regexp.QuoteMeta(pattern)
	}
	if q.Content && q.Insensitive {
		pattern = "(?i)" + pattern
	}
	var expression *regexp.Regexp
	var err error
	if q.Content {
		expression, err = regexp.Compile(pattern)
	} else {
		_, err = filepath.Match(pattern, "probe")
	}
	if err != nil {
		return Result{}, err
	}
	if q.Glob != "" {
		if _, err = filepath.Match(q.Glob, "probe"); err != nil {
			return Result{}, err
		}
	}
	if q.Content && !q.Recursive {
		info, err := r.Root.Stat(r.Path)
		if err != nil {
			return Result{}, err
		}
		if info.IsDir() {
			return Result{}, fmt.Errorf("directory search requires -r")
		}
	}
	var result Result
	var output strings.Builder
	stop := errors.New("query output limit reached")
	appendLine := func(text string) error {
		text = r.Redact(text)
		if result.Matches >= 1000 || output.Len()+len(text)+1 > 64*1024 {
			result.Truncated = true
			return stop
		}
		output.WriteString(text)
		output.WriteByte('\n')
		result.Matches++
		return nil
	}
	visited := 0
	err = fs.WalkDir(r.Root.FS(), r.Path, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			return walkErr
		}
		visited++
		if visited > 25000 {
			result.Truncated = true
			return stop
		}
		absolute := filepath.Join(r.Root.Name(), path)
		if absolute == r.Private || strings.HasPrefix(absolute, r.Private+string(filepath.Separator)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if path != r.Path && !q.Hidden && strings.HasPrefix(entry.Name(), ".") {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		relative, err := filepath.Rel(r.Path, path)
		if err != nil {
			return err
		}
		depth := 0
		if relative != "." {
			depth = len(strings.Split(relative, string(filepath.Separator)))
		}
		if q.MaxDepth >= 0 && depth > q.MaxDepth {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if !q.Content {
			if q.Files && !entry.Type().IsRegular() || q.Directories && !entry.IsDir() {
				return nil
			}
			found, err := filepath.Match(pattern, entry.Name())
			if err != nil {
				return err
			}
			if q.Glob != "" {
				found, err = filepath.Match(q.Glob, entry.Name())
			}
			if err != nil {
				return err
			}
			if found {
				return appendLine(absolute)
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if q.Glob != "" {
			found, err := filepath.Match(q.Glob, entry.Name())
			if err != nil || !found {
				return err
			}
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > fileops.MaxBytes {
			return nil
		}
		file, err := r.Root.Open(path)
		if err != nil {
			return err
		}
		return scan(ctx, file, absolute, expression, appendLine)
	})
	result.Text = output.String()
	if errors.Is(err, stop) {
		err = nil
	}
	return result, err
}

func scan(ctx context.Context, file *os.File, path string, pattern *regexp.Regexp, appendLine func(string) error) (err error) {
	defer func() { err = errors.Join(err, file.Close()) }()
	scanner := bufio.NewScanner(io.LimitReader(file, fileops.MaxBytes+1))
	scanner.Buffer(make([]byte, 4096), 256*1024)
	line := 0
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line++
		text := scanner.Text()
		if strings.ContainsRune(text, 0) {
			return nil
		}
		if pattern.MatchString(text) {
			if err := appendLine(fmt.Sprintf("%s:%d:%s", path, line, text)); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

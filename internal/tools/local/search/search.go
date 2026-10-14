// Purpose: Search confined file trees by filename or UTF-8 content with bounded output.
// Pattern: Filesystem search adapter.
// Usage: Local runtime calls Request.Execute with a resolved root after policy validation.
// Related: internal/tools/local/dispatch.go, internal/tools/local/file/limits.go.
package search

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
	resultfmt "github.com/keonho-kim/v-code/internal/tools/result"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Request struct {
	Root                   *os.Root
	Path, Private, Pattern string
	Content                bool
	Redact                 func(string) string
}

type match struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
	Text string `json:"text,omitempty"`
}

func (request Request) Execute(ctx context.Context) (toolapi.Result, error) {
	var expression *regexp.Regexp
	var err error
	if request.Content {
		expression, err = regexp.Compile(request.Pattern)
	} else {
		_, err = filepath.Match(request.Pattern, "probe")
	}
	if err != nil {
		return toolapi.Result{}, err
	}
	var matches []match
	visited, truncated := 0, false
	stop := errors.New("search limit reached")
	err = fs.WalkDir(request.Root.FS(), request.Path, func(path string, entry fs.DirEntry, walkErr error) error {
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
		absolute := filepath.Join(request.Root.Name(), path)
		if absolute == request.Private || strings.HasPrefix(absolute, request.Private+string(filepath.Separator)) {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		if !request.Content {
			found, err := filepath.Match(request.Pattern, entry.Name())
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
		if !info.Mode().IsRegular() || info.Size() > fileops.MaxBytes {
			return nil
		}
		file, err := request.Root.Open(path)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 4096), 256*1024)
		line := 0
		for scanner.Scan() {
			line++
			if expression.MatchString(scanner.Text()) {
				matches = append(matches, match{Path: absolute, Line: line, Text: request.Redact(scanner.Text())})
			}
			if len(matches) >= 1000 {
				truncated = true
				break
			}
		}
		return errors.Join(scanner.Err(), file.Close())
	})
	if err != nil && !errors.Is(err, stop) {
		return toolapi.Result{}, err
	}
	return resultfmt.Encode(struct {
		Matches   []match `json:"matches"`
		Truncated bool    `json:"truncated"`
	}{matches, truncated}, fmt.Sprintf("%d matches", len(matches)))
}

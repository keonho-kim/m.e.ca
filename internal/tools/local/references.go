// Purpose: Search bounded scoped file and directory names for input references.
// Pattern: Filesystem Adapter.
// Usage: Called by cancellable terminal completion and selection revalidation.
// Related: internal/tools/local/paths.go, internal/tools/local/references_test.go.
package local

import (
	"context"
	"errors"
	"github.com/keonho-kim/v-code/internal/policy"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

type Reference struct {
	Label, Path string
	Directory   bool
}

func (e *Executor) Reference(settings policy.Settings, path string) (Reference, error) {
	t, err := e.resolve(settings.Scope, path)
	if err != nil {
		return Reference{}, err
	}
	info, err := t.root.Stat(t.relative)
	if err != nil {
		return Reference{}, err
	}
	relative, err := filepath.Rel(e.project, t.absolute)
	if err != nil {
		return Reference{}, err
	}
	if relative == ".." || strings.HasPrefix(relative, "../") {
		relative = t.absolute
	}
	label := filepath.Base(t.absolute)
	if info.IsDir() {
		label += "/"
		relative += "/"
	}
	return Reference{Label: label, Path: relative, Directory: info.IsDir()}, nil
}
func (e *Executor) References(ctx context.Context, settings policy.Settings, token string) (out []Reference, err error) {
	base, query := ".", token
	pathMode := strings.Contains(token, "/")
	if pathMode {
		base = filepath.Dir(token)
		query = filepath.Base(token)
		if strings.HasSuffix(token, "/") {
			base = token
			query = ""
		}
	}
	root, err := e.resolve(settings.Scope, base)
	if err != nil {
		return nil, err
	}
	stop := errors.New("completion limit")
	visited := 0
	err = fs.WalkDir(root.root.FS(), root.relative, func(path string, entry fs.DirEntry, walkErr error) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if walkErr != nil {
			if path == root.relative {
				return walkErr
			}
			return nil
		}
		visited++
		if visited > 25000 || len(out) >= 100 {
			return stop
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if path != root.relative && entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "node_modules") {
			return fs.SkipDir
		}
		if pathMode && path != root.relative && filepath.Dir(path) != root.relative {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		absolute := filepath.Join(root.root.Name(), path)
		ref, resolveErr := e.Reference(settings, absolute)
		if resolveErr != nil {
			if entry.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if path == root.relative && (!pathMode || query != "") {
			return nil
		}
		if strings.Contains(strings.ToLower(entry.Name()), strings.ToLower(query)) {
			out = append(out, ref)
		}
		if pathMode && path != root.relative && entry.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
	if err != nil && !errors.Is(err, stop) {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := strings.HasPrefix(strings.ToLower(out[i].Label), strings.ToLower(query)), strings.HasPrefix(strings.ToLower(out[j].Label), strings.ToLower(query))
		if a != b {
			return a
		}
		return out[i].Path < out[j].Path
	})
	return out, nil
}

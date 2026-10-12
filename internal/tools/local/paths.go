// Purpose: Resolve real paths and capture filesystem versions through confined roots.
// Pattern: Filesystem Adapter.
// Usage: Used by preparation, revalidation and scoped file operations.
// Related: internal/tools/local/executor.go, internal/policy/path.go.

package local

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/keonho-kim/v-code/internal/policy"
)

const maxFileBytes = 4 * 1024 * 1024

type target struct {
	absolute, relative string
	root               *os.Root
}
type version struct {
	exists bool
	info   os.FileInfo
	digest [32]byte
}

func realPath(path string) (string, error) {
	var tail []string
	for {
		resolved, err := filepath.EvalSymlinks(path)
		if err == nil {
			for index := len(tail) - 1; index >= 0; index-- {
				resolved = filepath.Join(resolved, tail[index])
			}
			return resolved, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if info, statErr := os.Lstat(path); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", err
		}
		tail = append(tail, filepath.Base(path))
		path = parent
	}
}

func (executor *Executor) resolve(scope policy.Scope, path string) (target, error) {
	absolute, _, err := policy.Resolve(executor.project, executor.home, scope, path)
	if err != nil {
		return target{}, err
	}
	absolute, err = realPath(absolute)
	if err != nil {
		return target{}, err
	}
	if absolute == executor.private || strings.HasPrefix(absolute, executor.private+string(filepath.Separator)) {
		return target{}, fmt.Errorf("access to V credentials and state is denied")
	}
	_, relative, err := policy.Resolve(executor.project, executor.home, scope, absolute)
	if err != nil {
		return target{}, err
	}
	root := executor.projectRoot
	if scope == policy.System {
		root = executor.systemRoot
	}
	return target{absolute: absolute, relative: relative, root: root}, nil
}

func snapshot(path target) (captured version, err error) {
	info, err := path.root.Stat(path.relative)
	if errors.Is(err, os.ErrNotExist) {
		return version{}, nil
	}
	if err != nil {
		return captured, err
	}
	captured = version{exists: true, info: info}
	if info.IsDir() {
		return captured, nil
	}
	if !info.Mode().IsRegular() {
		return captured, fmt.Errorf("only regular files and directories are supported")
	}
	if info.Size() > maxFileBytes {
		return captured, fmt.Errorf("file exceeds modification size limit")
	}
	file, err := path.root.Open(path.relative)
	if err != nil {
		return captured, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	content, err := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	if err != nil {
		return captured, err
	}
	if len(content) > maxFileBytes {
		return captured, fmt.Errorf("file exceeds size limit")
	}
	captured.digest = sha256.Sum256(content)
	return captured, nil
}

func sameVersion(left, right version) bool {
	if left.exists != right.exists {
		return false
	}
	if !left.exists {
		return true
	}
	return os.SameFile(left.info, right.info) && left.info.Size() == right.info.Size() && left.info.ModTime().Equal(right.info.ModTime()) && left.digest == right.digest
}

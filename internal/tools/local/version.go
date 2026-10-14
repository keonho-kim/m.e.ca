// Purpose: Capture and compare the full file version bound to review and approval.
// Pattern: Filesystem version adapter.
// Usage: Preparation and execution guards call snapshot and sameVersion.
// Related: internal/tools/local/paths.go, internal/tools/local/operation.go.
package local

import (
	"crypto/sha256"
	"errors"
	"fmt"
	fileops "github.com/keonho-kim/v-code/internal/tools/local/file"
	"io"
	"os"
)

type version struct {
	exists bool
	info   os.FileInfo
	digest [32]byte
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
	if info.Size() > fileops.MaxBytes {
		return captured, fmt.Errorf("file exceeds modification size limit")
	}
	file, err := path.root.Open(path.relative)
	if err != nil {
		return captured, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	content, err := io.ReadAll(io.LimitReader(file, fileops.MaxBytes+1))
	if err != nil {
		return captured, err
	}
	if len(content) > fileops.MaxBytes {
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

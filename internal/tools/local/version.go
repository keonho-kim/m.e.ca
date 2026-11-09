// Purpose: Capture source and version once or stream a version-only execution guard.
// Pattern: Filesystem version adapter.
// Usage: Preparation and execution guards call snapshot and sameVersion.
// Related: internal/tools/local/candidate.go, internal/tools/local/operation.go.
package local

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	fileops "github.com/keonho-kim/meca/internal/tools/local/file"
)

type version struct {
	exists bool
	info   os.FileInfo
	digest [32]byte
}

func snapshot(path target) (captured version, err error) {
	captured, _, err = capture(path, false)
	return captured, err
}

// capture owns its source bytes; version-only checks retain only a bounded hash buffer.
func capture(path target, keepContent bool) (captured version, content []byte, err error) {
	info, err := path.root.Stat(path.relative)
	if errors.Is(err, os.ErrNotExist) {
		return version{}, nil, nil
	}
	if err != nil {
		return captured, nil, err
	}
	captured = version{exists: true, info: info}
	if info.IsDir() {
		return captured, nil, nil
	}
	if !info.Mode().IsRegular() {
		return captured, nil, faults.Errorf(diagcatalog.LocalOnlyRegularFilesAndDirectoriesAreSupported)
	}
	if info.Size() > fileops.MaxBytes {
		return captured, nil, faults.Errorf(diagcatalog.LocalFileExceedsModificationSizeLimit)
	}
	file, err := path.root.Open(path.relative)
	if err != nil {
		return captured, nil, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	var read int64
	if keepContent {
		content = make([]byte, int(info.Size())+1)
		n, readErr := io.ReadFull(file, content)
		if readErr != nil && !errors.Is(readErr, io.EOF) && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			return captured, nil, readErr
		}
		content, read = content[:n], int64(n)
		captured.digest = sha256.Sum256(content)
	} else {
		hash := sha256.New()
		read, err = io.Copy(hash, io.LimitReader(file, info.Size()+1))
		if err != nil {
			return captured, nil, err
		}
		hash.Sum(captured.digest[:0])
	}
	if read != info.Size() {
		return captured, nil, faults.Errorf(diagcatalog.LocalFileChangedWhileReadingItsVersion)
	}
	after, err := file.Stat()
	if err != nil {
		return captured, nil, err
	}
	current, err := path.root.Stat(path.relative)
	if err != nil {
		return captured, nil, err
	}
	if !os.SameFile(info, after) || !os.SameFile(after, current) || info.Size() != after.Size() || after.Size() != current.Size() || !info.ModTime().Equal(after.ModTime()) || !after.ModTime().Equal(current.ModTime()) {
		return captured, nil, faults.Errorf(diagcatalog.LocalFileChangedWhileCheckingItsVersion)
	}
	return captured, content, nil
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

func versionValue(v version) string {
	if !v.exists {
		return "missing"
	}
	if v.info.IsDir() {
		return "directory"
	}
	return fmt.Sprintf("%x", v.digest)
}

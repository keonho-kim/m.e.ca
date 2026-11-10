// Purpose: Append a sanitized live delta under the same retention and disk budget as full archives.
// Pattern: Executor-owned temporary file update.
// Usage: Projection owns the returned descriptor through excerpt reads and then closes it.
// Related: internal/tools/local/artifact_retention.go, internal/tools/local/command_incremental.go.
package local

import (
	"errors"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/capacity"
	"io"
	"os"
	"syscall"
)

func (e *Executor) appendArtifact(id, path string, body []byte, expected int64) (*os.File, error) {
	r := &e.retained
	r.mu.Lock()
	defer r.mu.Unlock()
	entry := r.entries[id]
	if entry == nil || entry.retired != nil || entry.files[path] != expected {
		return nil, faults.New(diagcatalog.Conflict, diagcatalog.LocalCapturedLogChangedBeforeItsDurableRecord)
	}
	maximum := r.limitBytes
	if maximum == 0 {
		maximum = maxArtifactBytes
	}
	for r.bytes+int64(len(body)) > maximum {
		if err := r.evict(); err != nil {
			return nil, err
		}
	}
	if err := capacity.CheckFree(e.artifacts, 1<<30+uint64(len(body))); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_RDWR|os.O_APPEND|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0600)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	if !info.Mode().IsRegular() || info.Size() != expected {
		return nil, errors.Join(faults.New(diagcatalog.Conflict, diagcatalog.LocalCapturedLogChangedBeforeItsDurableRecord), file.Close())
	}
	n, writeErr := file.Write(body)
	if writeErr == nil && n != len(body) {
		writeErr = io.ErrShortWrite
	}
	entry.files[path] += int64(n)
	r.bytes += int64(n)
	if writeErr != nil {
		return nil, errors.Join(writeErr, file.Close())
	}
	return file, nil
}

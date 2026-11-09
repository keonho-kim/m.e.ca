// Purpose: Mask newly appended command bytes once and update only the live log suffix.
// Pattern: Per-stream incremental masking with bounded excerpt reads.
// Usage: commandCaptures uses this path until capture rotation or redaction revision requires a full reset.
// Related: internal/tools/local/command_live.go, internal/tools/local/artifact_append.go.
package local

import (
	"errors"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/tools/local/process"
	"os"
)

func (e *Executor) projectIncrementalCapture(id, stream string, previous liveCapture, update process.Capture, mask *func(string, bool) string) (next liveCapture, resultErr error) {
	version := e.commandVersion()
	defer func() {
		if resultErr == nil && e.commandVersion() != version {
			resultErr = process.ErrProjectionBusy
		}
		if resultErr != nil {
			*mask = nil
			next = liveCapture{}
		}
	}()
	if update.Append {
		if *mask == nil || !previous.incremental || previous.version != version || update.Offset != previous.rawBytes || update.Total-update.Offset != int64(len(update.Text)) {
			return liveCapture{}, process.ErrProjectionBusy
		}
	} else {
		*mask = e.newChunkMask()
		previous = liveCapture{}
	}
	text := (*mask)(update.Text, false)
	next = liveCapture{revision: update.Revision, version: version, rawBytes: update.Total, incremental: true, bytes: previous.bytes + len(text), path: previous.path}
	if !update.Append {
		var err error
		next.excerpt = commandExcerpt(text)
		next.path, err = e.projectCaptureFile(id, stream, text)
		return next, err
	}
	if previous.path == "" {
		text = previous.excerpt + text
		next.excerpt = commandExcerpt(text)
		var err error
		next.path, err = e.projectCaptureFile(id, stream, text)
		return next, err
	}
	file, err := e.appendArtifact(id, previous.path, []byte(text), int64(previous.bytes))
	if err != nil {
		return previous, err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	before, err := file.Stat()
	if err != nil {
		return previous, err
	}
	var head, tail [outputExcerptBytes]byte
	count := min(next.bytes, outputExcerptBytes)
	if _, err := file.ReadAt(head[:count], 0); err != nil {
		return previous, err
	}
	if _, err := file.ReadAt(tail[:count], int64(next.bytes-count)); err != nil {
		return previous, err
	}
	after, err := os.Lstat(previous.path)
	if err != nil {
		return previous, err
	}
	if !os.SameFile(before, after) || after.Size() != int64(next.bytes) || !before.ModTime().Equal(after.ModTime()) {
		return previous, faults.New(diagcatalog.Conflict, diagcatalog.LocalCapturedLogChangedWhileReading)
	}
	if next.bytes <= outputExcerptBytes {
		next.excerpt = string(head[:count])
	} else {
		next.excerpt = commandExcerpt(string(head[:count]) + string(tail[:count]))
	}
	return next, nil
}

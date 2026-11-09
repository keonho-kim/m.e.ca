// Purpose: Bound private current-generation files across all tasks and release disk reservations.
// Pattern: Small memory prefix with a process-wide disk byte ledger.
// Usage: draftOriginal owns each draftText until reset or request completion.
// Related: internal/commands/server/draft_original.go, internal/storage/capacity/space_unix.go.
package server

import (
	"errors"
	"io"
	"os"

	"sync"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

var draftDisk struct {
	sync.Mutex
	used int64
}

const draftDiskBytes int64 = 512 << 20

type draftText struct {
	path           string
	memory         string
	file           *os.File
	total, charged int64
	failure        error
}

func (t *draftText) append(value string) {
	if int64(len(value)) == t.total {
		return
	}
	previous := t.total
	t.total = int64(len(value))
	if t.failure != nil {
		return
	}
	if t.file == nil && len(value) <= maxDraftBytes {
		t.memory = value
		return
	}
	next := int64(len(value)) - t.charged
	draftDisk.Lock()
	if next > draftDiskBytes-draftDisk.used {
		draftDisk.Unlock()
		t.fail(faults.New(diagcatalog.ResourceExhausted, diagcatalog.DetailSpoolBudget))
		return
	}
	draftDisk.used += next
	t.charged += next
	draftDisk.Unlock()
	if t.file == nil {
		if err := capacity.CheckFree(os.TempDir(), 1<<30+uint64(len(value))); err != nil {
			t.fail(err)
			return
		}
		file, err := os.CreateTemp("", "meca-draft-*")
		if err != nil {
			t.fail(err)
			return
		}
		t.file, t.path = file, file.Name()
		if err := os.Remove(t.path); err != nil {
			t.fail(err)
			return
		}
		t.path = ""
		t.memory = ""
		previous = 0
	}
	n, err := t.file.WriteAt([]byte(value[previous:]), previous)
	if err == nil && n != len(value)-int(previous) {
		err = io.ErrShortWrite
	}
	if err != nil {
		t.fail(err)
	}
}

// fail releases derived storage immediately; the original stream stays with
// the runtime and its durable journal, never with a failed cache reservation.
func (t *draftText) fail(cause error) {
	cleanup := t.release()
	t.memory = ""
	t.failure = errors.Join(cause, cleanup)
}

func (t *draftText) release() error {
	var err error
	if t.file != nil {
		err = t.file.Close()
		t.file = nil
	}
	if t.path != "" {
		removeErr := os.Remove(t.path)
		if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
			t.path = ""
		} else {
			err = errors.Join(err, removeErr)
		}
	}
	draftDisk.Lock()
	draftDisk.used -= t.charged
	t.charged = 0
	draftDisk.Unlock()
	return err
}

func (t *draftText) page(offset int64, limit int) (string, error) {
	if t.failure != nil {
		return "", t.failure
	}
	end := min(t.total, offset+int64(limit))
	if t.file == nil {
		return t.memory[offset:end], nil
	}
	body := make([]byte, int(end-offset))
	_, err := t.file.ReadAt(body, offset)
	return string(body), err
}

func (t *draftText) close() error {
	err := t.release()
	*t = draftText{}
	return err
}

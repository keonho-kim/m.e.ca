// Purpose: Own one process's private diagnostic files with bounded rotation and an exclusive live lease.
// Pattern: Serialized filesystem writer.
// Usage: TUI startup supplies File to Sink and closes it after the writer joins.
// Related: internal/diagnostics/logfile/retention.go, internal/config/statefs/root.go.
package logfile

import (
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"

	"github.com/keonho-kim/meca/internal/config/statefs"
	"github.com/keonho-kim/meca/internal/storage/capacity"
)

const maxFileBytes = 4 << 20
const retainedRuns = 8

type File struct {
	mu              sync.Mutex
	parent, root    *os.Root
	lease, file     *os.File
	name, directory string
	size            int64
	closed          bool
	current         atomic.Pointer[os.File]
	interrupted     atomic.Bool
}

func OpenFile(ctx context.Context, directory string) (value *File, err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parent, err := statefs.Open(directory)
	if err != nil {
		return nil, err
	}
	guard, err := statefs.Lock(ctx, parent)
	if err != nil {
		return nil, errors.Join(err, parent.Close())
	}
	defer func() {
		err = errors.Join(err, guard.Close())
		if err != nil && value != nil {
			err = errors.Join(err, value.Close())
			value = nil
		}
	}()
	if err := retainLogs(parent); err != nil {
		return nil, errors.Join(err, parent.Close())
	}
	if err := capacity.CheckFree(directory, (1<<30)+maxFileBytes); err != nil {
		return nil, errors.Join(err, parent.Close())
	}
	name := "run-" + rand.Text()
	if err := parent.Mkdir(name, 0700); err != nil {
		return nil, errors.Join(err, parent.Close())
	}
	root, err := parent.OpenRoot(name)
	if err != nil {
		return nil, errors.Join(err, parent.RemoveAll(name), parent.Close())
	}
	lease, err := statefs.Lease(root, true)
	if err != nil {
		return nil, errors.Join(err, root.Close(), parent.RemoveAll(name), parent.Close())
	}
	file, err := root.OpenFile("current.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, errors.Join(err, lease.Close(), root.Close(), parent.RemoveAll(name), parent.Close())
	}
	value = &File{parent: parent, root: root, lease: lease, file: file, name: name, directory: filepath.Join(directory, name)}
	value.current.Store(file)
	return value, nil
}

func (f *File) Write(body []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed || f.file == nil || f.interrupted.Load() {
		return 0, os.ErrClosed
	}
	if len(body) > maxFileBytes {
		return 0, fault.Errorf(catalog.DiagnosticRecordLimit)
	}
	if f.size+int64(len(body)) > maxFileBytes {
		if err := f.rotate(); err != nil {
			return 0, err
		}
	}
	n, err := f.file.Write(body)
	f.size += int64(n)
	return n, err
}

func (f *File) rotate() error {
	if f.interrupted.Load() {
		return os.ErrClosed
	}
	if err := capacity.CheckFree(f.directory, (1<<30)+maxFileBytes); err != nil {
		return err
	}
	f.current.Store(nil)
	if err := f.file.Close(); err != nil {
		f.file = nil
		return err
	}
	f.file = nil
	if err := f.root.Remove("previous-2.jsonl"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := f.root.Rename("previous-1.jsonl", "previous-2.jsonl"); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	if err := f.root.Rename("current.jsonl", "previous-1.jsonl"); err != nil {
		return err
	}
	file, err := f.root.OpenFile("current.jsonl", os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	f.file, f.size = file, 0
	f.current.Store(file)
	if f.interrupted.Load() {
		return errors.Join(os.ErrClosed, file.Close())
	}
	return nil
}

func (f *File) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	var err error
	if f.file != nil {
		err = f.file.Close()
		if errors.Is(err, os.ErrClosed) {
			err = nil
		}
		f.file = nil
		f.current.Store(nil)
	}
	err = errors.Join(err, f.lease.Close(), f.root.Close(), f.parent.Close())
	f.lease, f.root, f.parent = nil, nil, nil
	return err
}

// Interrupt closes the current owned descriptor without waiting for the writer's
// state mutex. Close still joins the writer before releasing roots and leases.
func (f *File) Interrupt() error {
	f.interrupted.Store(true)
	if file := f.current.Swap(nil); file != nil {
		err := file.Close()
		if !errors.Is(err, os.ErrClosed) {
			return err
		}
	}
	return nil
}

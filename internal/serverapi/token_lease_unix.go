//go:build linux || darwin

// Purpose: Reserve one server publisher for a token path without locking unrelated application state.
// Pattern: Caller-owned process-shared file lease.
// Usage: The server composition root holds this lease until its listener and workspaces stop.
// Related: cmd/meca/server.go, internal/serverapi/token.go.
package serverapi

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func LeaseTokenFile(path string) (*os.File, error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	file, err := root.OpenFile(filepath.Base(path)+".lock", os.O_CREATE|os.O_RDWR|syscall.O_NOFOLLOW, 0600)
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	if err := root.Close(); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	info, err := file.Stat()
	if err != nil {
		return nil, errors.Join(err, file.Close())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, errors.Join(fault.New(diag.FailedPrecondition, diag.ServerTokenLeasePrivate), file.Close())
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, errors.Join(fault.New(diag.Conflict, diag.ServerTokenPublisherOwned), file.Close())
		}
		return nil, errors.Join(fault.New(diag.FailedPrecondition, diag.ServerTokenLockFailed, err), file.Close())
	}
	// Keep the lock inode in place after close; unlinking it would split future owners.
	return file, nil
}

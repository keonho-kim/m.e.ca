//go:build linux || darwin

// Purpose: Hold one process owner per session database using Unix file locks.
// Pattern: Resource lifecycle boundary.
// Usage: cmd/meca closes the session lease after task workers stop.
// Related: internal/storage/sqlite/lease_test.go, internal/storage/sqlite/schema_test.go, cmd/meca/main.go.
package sqlite

import (
	"errors"

	"os"
	"path/filepath"
	"syscall"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// LeaseServer reserves durable request status ownership for one canonical project.
	// The composition root keeps this lease until all server work has joined.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type SessionLease struct{ file *os.File }

func (s *Sessions) LeaseServer(project string) (*SessionLease, error) {
	directory, _, err := s.projectDirectory(project, true)
	if err != nil {
		return nil, err
	}
	file, err := openLock(filepath.Join(directory, ".server.lock"), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, faults.Errorf(diagcatalog.SqliteServerOwnershipUnavailableAnotherMECAServerMay, err)
	}
	return &SessionLease{file: file}, nil
}

func (s *Store) LeaseSession() (*SessionLease, error) {
	file, err := openLock(s.path+".lock", syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, faults.Errorf(diagcatalog.SqliteConversationLockUnavailableAnotherMECAProcessMay, err)
	}
	return &SessionLease{file: file}, nil
}
func (l *SessionLease) Close() error { return l.file.Close() }
func openLock(path string, flags int) (*os.File, error) {
	// Private state is owned and checked by bootstrap. Reject a replaced leaf symlink.
	fd, err := syscall.Open(path, syscall.O_CREAT|syscall.O_RDWR|syscall.O_NOFOLLOW|syscall.O_CLOEXEC, 0600)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), path)
	if err = syscall.Flock(int(file.Fd()), flags); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return file, nil
}

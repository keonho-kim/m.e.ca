//go:build linux || darwin

// Purpose: Serialize DB bootstrap and hold one process owner per session using Unix file locks.
// Pattern: Resource lifecycle boundary.
// Usage: Store.Open releases its bootstrap lock; cmd/v closes the session lease after workers stop.
// Related: internal/storage/sqlite/lease_test.go, internal/storage/sqlite/schema_test.go, cmd/v/main.go.
package sqlite

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

type SessionLease struct{ file *os.File }

func (s *Store) LeaseSession(session string) (*SessionLease, error) {
	if session == "" || len(session) > 128 {
		return nil, fmt.Errorf("invalid session ID")
	}
	digest := sha256.Sum256([]byte(session))
	name := fmt.Sprintf(".session-%x.lock", digest[:12])
	file, err := openLock(filepath.Join(s.directory, name), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		return nil, fmt.Errorf("conversation lock unavailable; another V process may own this session: %w", err)
	}
	return &SessionLease{file: file}, nil
}
func (l *SessionLease) Close() error { return l.file.Close() }
func (s *Store) schemaLock() (*os.File, error) {
	return openLock(filepath.Join(s.directory, ".schema.lock"), syscall.LOCK_EX)
}
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

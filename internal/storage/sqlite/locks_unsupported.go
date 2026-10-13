//go:build !linux && !darwin

// Purpose: Reject state locking outside the supported Unix platforms.
// Pattern: Platform boundary.
// Usage: Used by database or session startup on unsupported targets.
// Related: internal/storage/sqlite/locks_unix.go, cmd/v/main.go.
package sqlite

import (
	"fmt"
	"os"
)

type SessionLease struct{}

func (*Store) LeaseSession(string) (*SessionLease, error) {
	return nil, fmt.Errorf("session locking requires Linux or macOS")
}
func (*SessionLease) Close() error { return fmt.Errorf("session locking requires Linux or macOS") }
func (*Store) schemaLock() (*os.File, error) {
	return nil, fmt.Errorf("database initialization requires Linux or macOS")
}

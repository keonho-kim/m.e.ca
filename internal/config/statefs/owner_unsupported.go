//go:build !linux && !darwin

// Purpose: Reject private state initialization on unverified platforms.
// Pattern: Platform boundary.
// Usage: Used by configuration code outside Linux and macOS.
// Related: internal/config/statefs/owner_unix.go, cmd/v/main.go.
package statefs

import (
	"fmt"
	"os"
)

func checkOwner(os.FileInfo) error {
	return fmt.Errorf("private state is supported only on Linux and macOS")
}

// Purpose: Open and verify the private application directory owned by the current user.
// Pattern: Filesystem trust boundary.
// Usage: Used by config, prompts and bootstrap before accessing ~/.v files.
// Related: internal/config/statefs/files.go, internal/config/statefs/owner_unix.go, internal/config/config.go.
package statefs

import (
	"errors"
	"fmt"
	"os"
)

func Open(directory string) (*os.Root, error) {
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return nil, err
	}
	info, err := os.Lstat(directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("application state must be a real directory")
	}
	if err := checkOwner(info); err != nil {
		return nil, err
	}
	root, err := os.OpenRoot(directory)
	if err != nil {
		return nil, err
	}
	file, err := root.Open(".")
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	current, statErr := file.Stat()
	if statErr != nil {
		return nil, errors.Join(statErr, file.Close(), root.Close())
	}
	if !os.SameFile(info, current) {
		return nil, errors.Join(fmt.Errorf("application directory changed during initialization"), file.Close(), root.Close())
	}
	if err = errors.Join(file.Chmod(0700), file.Close()); err != nil {
		return nil, errors.Join(err, root.Close())
	}
	return root, nil
}

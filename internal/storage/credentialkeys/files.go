// Purpose: Enforce ownership and private filesystem access for local encryption-key storage.
// Pattern: Filesystem trust boundary with descriptor identity checks.
// Usage: The key database adapter validates directories and files before SQLite access.
// Related: internal/storage/credentialkeys/database.go, internal/config/statefs/root.go.
package credentialkeys

import (
	"errors"
	"os"
	"syscall"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (s Store) root(create bool) (*os.Root, error) {
	if create {
		return statefs.Open(s.Directory)
	}
	info, err := os.Lstat(s.Directory)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode().Perm()&0077 != 0 || !owned(info) {
		return nil, fault.Errorf(diag.LocalKeysPrivateDirectory)
	}
	root, err := os.OpenRoot(s.Directory)
	if err != nil {
		return nil, err
	}
	file, err := root.Open(".")
	if err != nil {
		return nil, errors.Join(err, root.Close())
	}
	current, statErr := file.Stat()
	if err := errors.Join(statErr, file.Close()); err != nil {
		return nil, errors.Join(err, root.Close())
	}
	if !os.SameFile(info, current) || current.Mode().Perm()&0077 != 0 || !owned(current) {
		return nil, errors.Join(fault.Errorf(diag.LocalKeysPrivateDirectory), root.Close())
	}
	return root, nil
}

func privateFile(root *os.Root, name string) (resultErr error) {
	file, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || !owned(info) {
		return fault.Errorf(diag.LocalKeysPrivateFile)
	}
	return nil
}

func owned(info os.FileInfo) bool {
	stat, ok := info.Sys().(*syscall.Stat_t)
	return ok && stat.Uid == uint32(os.Geteuid())
}

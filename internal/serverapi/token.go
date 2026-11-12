// Purpose: Generate credentials and atomically publish them only after listener preparation.
// Pattern: Credential filesystem boundary.
// Usage: cmd/meca prepares one token for masking and publishes it after a successful bind.
// Related: cmd/meca/server.go, internal/config/statefs/files.go.
package serverapi

import (
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func validToken(token string) bool {
	if len(token) < 32 || len(token) > 256 {
		return false
	}
	for _, r := range token {
		if r < 33 || r > 126 {
			return false
		}
	}
	return true
}

func NewToken() string { return rand.Text() + rand.Text() }

func PublishToken(path, token string) (resultErr error) {
	if !validToken(token) {
		return faults.Errorf(diagcatalog.ServerapiTokenFileMustContainToPrintableNon)
	}
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	name := filepath.Base(path)
	before, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return statefs.Replace(root, name, []byte(token+"\n"))
	} else if err != nil {
		return err
	}
	if !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 {
		return faults.Errorf(diagcatalog.ServerapiTokenFileMustBeAPrivateRegular)
	}
	if stat, ok := before.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Geteuid()) {
		return faults.Errorf(diagcatalog.ServerapiTokenFileMustBeOwnedByThe)
	}
	file, err := root.Open(name)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	after, err := file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(before, after) {
		return faults.Errorf(diagcatalog.ServerapiTokenFileChangedWhileOpening)
	}
	body, err := io.ReadAll(io.LimitReader(file, 258))
	if err != nil {
		return err
	}
	if !validToken(strings.TrimSuffix(string(body), "\n")) {
		return faults.Errorf(diagcatalog.ServerapiTokenFileMustContainToPrintableNon)
	}
	// Never reinterpret an unrelated existing private file as a token target.
	// Keep the old token intact until the replacement is complete and synced.
	return statefs.Replace(root, name, []byte(token+"\n"))
}

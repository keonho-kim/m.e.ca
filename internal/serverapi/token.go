// Purpose: Atomically create or read a private server token without exposing its contents.
// Pattern: Credential filesystem boundary.
// Usage: cmd/meca loads one token for both masking and listener authentication.
// Related: cmd/meca/server.go, internal/config/statefs/files.go.
package serverapi

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/keonho-kim/meca/internal/config/statefs"
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

func LoadToken(path string) (token string, resultErr error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, root.Close()) }()
	name := filepath.Base(path)
	if _, err := root.Lstat(name); errors.Is(err, os.ErrNotExist) {
		if err := statefs.PublishMissing(root, name, []byte(rand.Text()+rand.Text()+"\n")); err != nil {
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	before, err := root.Lstat(name)
	if err != nil {
		return "", err
	}
	if !before.Mode().IsRegular() || before.Mode().Perm()&0077 != 0 {
		return "", fmt.Errorf("token file must be a private regular file (0600)")
	}
	if stat, ok := before.Sys().(*syscall.Stat_t); !ok || stat.Uid != uint32(os.Geteuid()) {
		return "", fmt.Errorf("token file must be owned by the current user")
	}
	file, err := root.Open(name)
	if err != nil {
		return "", err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	after, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !os.SameFile(before, after) {
		return "", fmt.Errorf("token file changed while opening")
	}
	body, err := io.ReadAll(io.LimitReader(file, 258))
	if err != nil {
		return "", err
	}
	token = strings.TrimSuffix(string(body), "\n")
	if !validToken(token) {
		return "", fmt.Errorf("token file must contain 32 to 256 printable non-space ASCII bytes")
	}
	return token, nil
}

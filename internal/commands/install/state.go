// Purpose: Validate and persist installation ownership under private application state.
// Pattern: Installation receipt adapter.
// Usage: Install and Uninstall share binary and shell-registration records.
// Related: internal/commands/install/install.go, internal/commands/install/uninstall.go.
package install

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/config/statefs"
	"io"
	"os"
	"path/filepath"
)

const receiptName = "installation.json"
const binaryName = ".local/bin/meca"
const maxFileBytes = 16 * 1024 * 1024

type registration struct {
	Name      string `json:"name"`
	Created   bool   `json:"created"`
	Separator bool   `json:"separator"`
}
type receipt struct {
	Digest   string         `json:"binary_sha256"`
	Profiles []registration `json:"shell_profiles"`
}

func loadReceipt(root *os.Root) (r receipt, err error) {
	body, exists, _, err := readFile(root, statefs.DirectoryName+"/"+receiptName)
	if err != nil || !exists {
		return r, err
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if err := d.Decode(&r); err != nil {
		return r, fmt.Errorf("invalid installation receipt: %w", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return r, fmt.Errorf("trailing installation receipt content")
	}
	digest, err := hex.DecodeString(r.Digest)
	if err != nil || len(digest) != 32 {
		return r, fmt.Errorf("invalid installed binary digest")
	}
	seen := map[string]bool{}
	for _, profile := range r.Profiles {
		if !validProfile(profile.Name) || seen[profile.Name] {
			return r, fmt.Errorf("invalid shell registration")
		}
		seen[profile.Name] = true
	}
	return r, nil
}

func saveReceipt(home string, r receipt) error {
	body, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return statefs.Write(filepath.Join(home, statefs.DirectoryName), receiptName, body)
}

func readFile(root *os.Root, name string) ([]byte, bool, os.FileMode, error) {
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, 0600, nil
	}
	if err != nil {
		return nil, false, 0, err
	}
	if !info.Mode().IsRegular() || info.Size() > maxFileBytes {
		return nil, false, 0, fmt.Errorf("%s must be a regular file within 16 MiB", name)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, false, 0, err
	}
	body, readErr := io.ReadAll(io.LimitReader(file, maxFileBytes+1))
	err = errors.Join(readErr, file.Close())
	if len(body) > maxFileBytes {
		err = errors.Join(err, fmt.Errorf("%s exceeds 16 MiB", name))
	}
	return body, true, info.Mode().Perm(), err
}

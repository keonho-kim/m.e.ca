// Purpose: Replace installed prompt files with the release defaults while avoiding identical writes.
// Pattern: Filesystem synchronization boundary.
// Usage: Install invokes Refresh under its exclusive application-state lease.
// Related: internal/config/prompts/prompts.go, internal/commands/install/install.go, internal/config/statefs/files.go.
package prompts

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/keonho-kim/meca/internal/agent/prompt"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

func Refresh(root *os.Root) (err error) {
	if err := prepareDirectory(root); err != nil {
		return err
	}
	directory, err := root.OpenRoot("prompts")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, directory.Close()) }()
	defaults := prompt.Defaults()
	for name, text := range defaults.Files() {
		body := []byte(*text)
		matches, err := matchesDefault(directory, name, body)
		if err != nil {
			return err
		}
		if !matches {
			if err := statefs.Write(directory.Name(), name, body); err != nil {
				return err
			}
		}
	}
	return nil
}

func matchesDefault(directory *os.Root, name string, expected []byte) (matches bool, err error) {
	info, err := directory.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("%s must be a regular prompt file", name)
	}
	if info.Size() != int64(len(expected)) {
		return false, nil
	}
	file, err := directory.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	body, err := io.ReadAll(io.LimitReader(file, int64(len(expected))+1))
	return bytes.Equal(body, expected), err
}

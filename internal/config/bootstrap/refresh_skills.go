// Purpose: Refresh release-owned skill files without rewriting identical private content.
// Pattern: Filesystem synchronization boundary.
// Usage: Install calls RefreshSkills under its exclusive application-state lease.
// Related: internal/config/bootstrap/skills.go, internal/config/statefs/files.go, internal/commands/install/install.go.
package bootstrap

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"

	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// RefreshSkills replaces bundled files while retaining paths outside the release manifest.
func RefreshSkills(ctx context.Context, root *os.Root) error {
	return bootstrapSkills(ctx, root, true)
}

func refreshSkillFile(root *os.Root, name string, content []byte) error {
	matches, err := skillFileMatches(root, name, content)
	if err != nil || matches {
		return err
	}
	return statefs.Replace(root, name, content)
}

func skillFileMatches(root *os.Root, name string, expected []byte) (matches bool, err error) {
	info, err := root.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, faults.New(diagcatalog.FailedPrecondition, diagcatalog.BootstrapSkillFileMustBeRegular, name)
	}
	const permissionBits = os.ModePerm | os.ModeSetuid | os.ModeSetgid | os.ModeSticky
	if info.Size() != int64(len(expected)) || info.Mode()&permissionBits != 0600 {
		return false, nil
	}
	file, err := root.Open(name)
	if err != nil {
		return false, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	current, err := file.Stat()
	if err != nil {
		return false, err
	}
	if !os.SameFile(info, current) {
		return false, faults.Errorf(diagcatalog.BootstrapChangedDuringInitialization, name)
	}
	body, err := io.ReadAll(io.LimitReader(file, int64(len(expected))+1))
	return bytes.Equal(body, expected), err
}

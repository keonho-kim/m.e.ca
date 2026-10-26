// Purpose: Stage a verified release and hand installation to its executable with guaranteed temporary cleanup.
// Pattern: Upgrade use case with explicit resource ownership.
// Usage: The maintenance CLI calls Run with one authenticated release client.
// Related: internal/commands/upgrade/release.go, internal/commands/install/install.go, cmd/meca/upgrade.go.
package upgrade

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/config/statefs"
)

type Result struct {
	Version        string
	AlreadyCurrent bool
}

func Run(ctx context.Context, home string, client *Client) (result Result, err error) {
	installed := install.BinaryPath(home)
	info, err := os.Lstat(installed)
	if err != nil {
		return result, fmt.Errorf("read installed MECA binary: %w; run meca install before upgrade", err)
	}
	if !info.Mode().IsRegular() {
		return result, fmt.Errorf("upgrade requires an installed regular MECA binary; run meca install first")
	}
	root, err := statefs.Open(filepath.Join(home, statefs.DirectoryName))
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	name := ".upgrade-" + rand.Text()
	if err := root.Mkdir(name, 0700); err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, root.RemoveAll(name)) }()
	stage, err := root.OpenRoot(name)
	if err != nil {
		return result, err
	}
	defer func() { err = errors.Join(err, stage.Close()) }()
	result.Version, err = client.fetch(ctx, stage)
	if err != nil {
		return result, err
	}
	candidate := filepath.Join(stage.Name(), "meca")
	previous, err := executableHash(installed)
	if err != nil {
		return result, err
	}
	next, err := executableHash(candidate)
	if err != nil {
		return result, err
	}
	result.AlreadyCurrent = previous == next
	// Reinstall even identical bytes: the release owns the default prompt files.
	command := exec.CommandContext(ctx, candidate, "install", "--no-shell")
	command.Env = append(os.Environ(), "HOME="+home)
	command.WaitDelay = 2 * time.Second
	output, err := command.CombinedOutput()
	if err != nil {
		return result, fmt.Errorf("release %s installation failed: %w: %s", result.Version, err, strings.TrimSpace(string(output)))
	}
	return result, nil
}

func executableHash(path string) (digest [sha256.Size]byte, err error) {
	file, err := os.Open(path)
	if err != nil {
		return digest, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	hash := sha256.New()
	count, err := io.Copy(hash, io.LimitReader(file, maxArchiveBytes+1))
	if count > maxArchiveBytes {
		return digest, fmt.Errorf("MECA executable exceeds 256 MiB")
	}
	copy(digest[:], hash.Sum(nil))
	return digest, err
}

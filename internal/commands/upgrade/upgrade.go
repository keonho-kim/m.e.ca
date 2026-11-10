// Purpose: Stage a verified release and hand installation to its executable with guaranteed temporary cleanup.
// Pattern: Upgrade use case with explicit resource ownership.
// Usage: The maintenance CLI passes its selected distribution and authenticated release client to Run.
// Related: internal/commands/upgrade/release.go, internal/commands/install/install.go, cmd/meca/upgrade.go.
package upgrade

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"

	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/keonho-kim/meca/internal/commands/install"
	"github.com/keonho-kim/meca/internal/config/statefs"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/distribution"
)

type Result struct {
	Version        string
	AlreadyCurrent bool
}

func Run(ctx context.Context, home string, client *Client, kind distribution.Kind) (result Result, err error) {
	installed := install.BinaryPath(home)
	info, err := os.Lstat(installed)
	if err != nil {
		return result, faults.Errorf(diagcatalog.UpgradeReadInstalledMECABinaryRunMecaInstall, err)
	}
	if !info.Mode().IsRegular() {
		return result, faults.Errorf(diagcatalog.UpgradeUpgradeRequiresAnInstalledRegularMECABinary)
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
	result.Version, err = client.fetch(ctx, stage, kind)
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
		return result, faults.Errorf(diagcatalog.UpgradeReleaseInstallationFailed, result.Version, err, strings.TrimSpace(string(output)))
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
		return digest, faults.Errorf(diagcatalog.UpgradeMECAExecutableExceedsMiB)
	}
	copy(digest[:], hash.Sum(nil))
	return digest, err
}

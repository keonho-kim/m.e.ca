// Purpose: Publish MECA's executable and verify ownership before replacement or removal.
// Pattern: Filesystem adapter.
// Usage: Installation copies one binary into the user's local bin directory.
// Related: internal/commands/install/install.go, internal/commands/install/state.go.
package install

import (
	"crypto/rand"
	"crypto/sha256"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

	// BinaryPath is the user-owned installation location shared by install and upgrade.
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxBinaryBytes = 256 * 1024 * 1024

func BinaryPath(home string) string { return filepath.Join(home, binaryName) }

func binaryDigest(path string) (digest string, err error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() > maxBinaryBytes {
		return "", faults.Errorf(diagcatalog.InstallExecutableMustBeARegularFileWithin)
	}
	hash := sha256.New()
	size, err := io.Copy(hash, io.LimitReader(file, maxBinaryBytes+1))
	if err != nil || size > maxBinaryBytes {
		return "", errors.Join(err, faults.Errorf(diagcatalog.InstallCouldNotHashBoundedExecutable))
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func checkInstalledBinary(root *os.Root, expected string) error {
	info, err := root.Lstat(binaryName)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return faults.Errorf(diagcatalog.InstallInstalledMecaMustBeARegularFile)
	}
	path := filepath.Join(root.Name(), binaryName)
	digest, err := binaryDigest(path)
	if err != nil {
		return err
	}
	if expected != "" && digest == expected {
		return nil
	}
	// Interrupted updates may leave a different MECA build than the recorded one.
	build, err := buildinfo.ReadFile(path)
	if err == nil && build.Path == "github.com/keonho-kim/meca/cmd/meca" && build.Main.Path == "github.com/keonho-kim/meca" {
		return nil
	}
	return faults.Errorf(diagcatalog.InstallBelongsToAnotherProgramOrWasReplaced, path)
}

func copyBinary(root *os.Root, source, expected string) (err error) {
	if err := root.MkdirAll(".local/bin", 0755); err != nil {
		return err
	}
	name := ".local/bin/.meca-install-" + rand.Text()
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := root.Remove(name); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	input, openErr := os.Open(source)
	if openErr != nil {
		return errors.Join(openErr, file.Close())
	}
	hash := sha256.New()
	size, copyErr := io.Copy(io.MultiWriter(file, hash), io.LimitReader(input, maxBinaryBytes+1))
	err = errors.Join(copyErr, input.Close(), file.Sync(), file.Close())
	if err != nil {
		return err
	}
	if size > maxBinaryBytes || fmt.Sprintf("%x", hash.Sum(nil)) != expected {
		return faults.Errorf(diagcatalog.InstallExecutableChangedDuringInstallation)
	}
	return root.Rename(name, binaryName)
}

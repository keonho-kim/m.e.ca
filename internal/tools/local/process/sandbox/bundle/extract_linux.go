//go:build linux

// Purpose: Extract and verify the architecture-matched sandbox helper embedded in Linux releases.
// Pattern: Private executable resource lifecycle.
// Usage: Linux sandbox preparation calls Executable; no runtime download or unrestricted fallback occurs.
// Related: scripts/build-bwrap.sh, internal/tools/local/process/sandbox/linux.go.
package bundle

import (
	"bytes"
	"crypto/sha256"
	"debug/elf"
	"embed"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/config/statefs"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

//go:embed assets
var resources embed.FS

func Executable(private string) (path string, err error) {
	payload, err := resources.ReadFile("assets/" + runtime.GOARCH + "/bwrap")
	if err != nil {
		return "", fmt.Errorf("this Linux build has no embedded %s sandbox helper; rebuild with scripts/build-bwrap.sh: %w", runtime.GOARCH, err)
	}
	binary, err := elf.NewFile(bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	machine := binary.Machine
	if err := binary.Close(); err != nil {
		return "", err
	}
	if runtime.GOARCH == "amd64" && machine != elf.EM_X86_64 || runtime.GOARCH == "arm64" && machine != elf.EM_AARCH64 {
		return "", fmt.Errorf("embedded sandbox architecture mismatch")
	}
	digest := sha256.Sum256(payload)
	directory := filepath.Join(private, "bin", fmt.Sprintf("bwrap-%x", digest[:]))
	root, err := statefs.Open(directory)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, root.Close()) }()
	if err := statefs.PublishMissing(root, "bwrap", payload); err != nil {
		return "", err
	}
	license, err := resources.ReadFile("assets/" + runtime.GOARCH + "/COPYING")
	if err != nil {
		return "", err
	}
	if err := statefs.PublishMissing(root, "COPYING", license); err != nil {
		return "", err
	}
	info, err := root.Lstat("bwrap")
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() || info.Size() != int64(len(payload)) {
		return "", fmt.Errorf("sandbox helper is not the expected regular executable")
	}
	file, err := root.Open("bwrap")
	if err != nil {
		return "", err
	}
	observed, readErr := io.ReadAll(io.LimitReader(file, int64(len(payload))+1))
	if err := errors.Join(readErr, file.Close()); err != nil {
		return "", err
	}
	if sha256.Sum256(observed) != digest {
		return "", fmt.Errorf("sandbox helper checksum mismatch; refusing execution")
	}
	if err := root.Chmod("bwrap", os.FileMode(0700)); err != nil {
		return "", err
	}
	return filepath.Join(directory, "bwrap"), nil
}

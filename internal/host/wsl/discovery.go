// Purpose: Locate Windows PowerShell when WSL's imported Windows PATH is absent.
// Pattern: Bounded mount-table path discovery.
// Usage: Clipboard input and desktop folder selection share this Windows host discovery.
// Related: internal/tui/clipboard/wsl.go, internal/desktop/folderpicker/picker_linux.go.
package wsl

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func readWSLMounts() ([]byte, error) {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, 65537))
	if len(data) > 65536 {
		return nil, os.ErrInvalid
	}
	return data, err
}

func findPowerShell(lookup func(string) (string, error), mounts func() ([]byte, error)) (string, error) {
	if path, err := lookup("powershell.exe"); err == nil {
		return path, nil
	}
	data, err := mounts()
	if err != nil {
		return "", err
	}
	for _, candidate := range powerShellCandidates(string(data)) {
		if path, err := lookup(candidate); err == nil {
			return path, nil
		}
	}
	return "", os.ErrNotExist
}

func powerShellCandidates(mounts string) []string {
	decode := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	var paths []string
	for _, line := range strings.Split(mounts, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		if fields[2] != "drvfs" && !(fields[2] == "9p" && strings.Contains(fields[3], "aname=drvfs")) {
			continue
		}
		root := decode.Replace(fields[1])
		if !filepath.IsAbs(root) {
			continue
		}
		paths = append(paths, filepath.Join(root, "Windows", "System32", "WindowsPowerShell", "v1.0", "powershell.exe"))
	}
	return paths
}

// PowerShell finds the built-in Windows runtime without requiring imported Windows PATH.
func PowerShell() (string, error) { return findPowerShell(exec.LookPath, readWSLMounts) }

// Available identifies the Windows-hosted Linux environment without starting a subprocess.
func Available() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	if os.Getenv("WSL_INTEROP") != "" || os.Getenv("WSL_DISTRO_NAME") != "" {
		return true
	}
	data, err := os.ReadFile("/proc/sys/kernel/osrelease")
	return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
}

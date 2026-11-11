// Purpose: Locate Windows PowerShell when WSL's imported Windows PATH is absent.
// Pattern: Bounded mount-table path discovery.
// Usage: The WSL image adapter prefers PATH, then mounted Windows system directories.
// Related: internal/tui/clipboard/wsl.go.
package clipboard

import (
	"io"
	"os"
	"path/filepath"
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

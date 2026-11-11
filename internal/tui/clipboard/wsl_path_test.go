// Purpose: Verify WSL mount-based discovery without requiring a Windows host.
// Pattern: Pure path selection contracts.
// Usage: go test ./internal/tui/clipboard -run TestPowerShell.
// Related: internal/tui/clipboard/wsl_path.go.
package clipboard

import (
	"os"
	"reflect"
	"testing"
)

func TestPowerShellCustomMountRootAndEscapes(t *testing.T) {
	mounts := "C:\\ /windows/c 9p rw,aname=drvfs;path=C:\\ 0 0\nD:\\ /disk\\040d drvfs rw 0 0\ntmpfs /run tmpfs rw 0 0"
	want := []string{"/windows/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe", "/disk d/Windows/System32/WindowsPowerShell/v1.0/powershell.exe"}
	if got := powerShellCandidates(mounts); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	lookup := func(path string) (string, error) {
		if path == want[1] {
			return path, nil
		}
		return "", os.ErrNotExist
	}
	got, err := findPowerShell(lookup, func() ([]byte, error) { return []byte(mounts), nil })
	if err != nil || got != want[1] {
		t.Fatal(got, err)
	}
}

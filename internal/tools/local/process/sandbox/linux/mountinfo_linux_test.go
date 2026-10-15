//go:build linux

// Purpose: Verify escaped mount paths and path boundaries used by runtime projection.
// Pattern: Kernel protocol contract tests.
// Usage: go test ./internal/tools/local/process/sandbox/linux.
// Related: internal/tools/local/process/sandbox/linux/mountinfo_linux.go.
package linux

import (
	"strings"
	"testing"
)

func TestMountPoints(t *testing.T) {
	source := "1 0 0:1 / / rw - ext4 /dev/root rw\n2 1 0:2 / /usr/lib/wsl\\040data rw - tmpfs tmpfs rw\n"
	points, err := mountPoints(strings.NewReader(source))
	if err != nil || len(points) != 2 || points[1] != "/usr/lib/wsl data" {
		t.Fatalf("%v %v", points, err)
	}
	if within("/usr/lib", "/usr/library") {
		t.Fatal("prefix confused with path containment")
	}
	for _, input := range []string{"broken", "2 1 0:2 / relative rw - tmpfs tmpfs rw", "2 1 0:2 / /bad\\xyz rw - tmpfs tmpfs rw"} {
		if _, err := mountPoints(strings.NewReader(input)); err == nil {
			t.Fatalf("accepted malformed record %q", input)
		}
	}
}

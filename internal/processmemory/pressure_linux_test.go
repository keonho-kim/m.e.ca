// Purpose: Verify bounded Linux available-memory parsing for browser admission.
// Pattern: Pure procfs fixture tests.
// Usage: go test ./internal/processmemory on Linux.
// Related: internal/processmemory/pressure_linux.go.
package processmemory

import (
	"strings"
	"testing"
)

func TestAvailableMemory(t *testing.T) {
	value, err := availableMemory(strings.NewReader("MemTotal: 9000 kB\nMemAvailable: 2048 kB\n"))
	if err != nil || value != 2<<20 {
		t.Fatal(value, err)
	}
	for _, text := range []string{"", "MemAvailable: bad kB", "MemAvailable: 18446744073709551615 kB", "MemFree: 100 kB"} {
		if _, err := availableMemory(strings.NewReader(text)); err == nil {
			t.Fatal("invalid memory snapshot accepted", text)
		}
	}
}

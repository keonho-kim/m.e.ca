//go:build linux

// Purpose: Verify kernel prerequisites are mandatory and cannot select an unrestricted fallback.
// Pattern: Kernel contract test.
// Usage: go test ./internal/tools/local/process/sandbox/linux.
// Related: internal/tools/local/process/sandbox/linux/landlock_linux.go.
package linux

import "testing"

func TestRequiredLandlockABI(t *testing.T) {
	for _, abi := range []int{0, 1, 2} {
		if err := requireABI(abi); err == nil {
			t.Fatalf("accepted incomplete filesystem restrictions at ABI %d", abi)
		}
	}
	if err := requireABI(3); err != nil {
		t.Fatal(err)
	}
}

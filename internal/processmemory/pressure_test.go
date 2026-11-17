// Purpose: Verify native pressure observation and an optional constrained-container expectation.
// Pattern: Platform smoke test with explicit environment-controlled integration assertion.
// Usage: MECA_TEST_MEMORY_PRESSURE=1 in a memory-limited Linux container asserts admission pressure.
// Related: internal/processmemory/pressure_linux.go, internal/processmemory/pressure_darwin.go.
package processmemory

import (
	"os"
	"runtime"
	"testing"
)

func TestPressureObservation(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("native pressure unavailable")
	}
	pressure, err := Pressure()
	if err != nil {
		t.Fatal(err)
	}
	if os.Getenv("MECA_TEST_MEMORY_PRESSURE") == "1" && !pressure {
		t.Fatal("constrained memory not detected")
	}
	t.Log("memory pressure:", pressure)
}

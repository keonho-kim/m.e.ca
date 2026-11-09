// Purpose: Verify that direct error and logger declarations are rejected.
// Pattern: Analyzer contract test.
// Usage: go -C devtools test ./internal/diagnosticcheck.
// Related: devtools/internal/diagnosticcheck/check.go, devtools/internal/diagnosticcheck/testdata.
package diagnosticcheck

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestCentralDeclarations(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), Analyzer, "example")
}

// Purpose: Enforce centralized product error messages, codes and logging boundaries.
// Pattern: Go analysis command.
// Usage: go -C devtools run ./cmd/diagnosticscheck -C=.. ./...
// Related: devtools/internal/diagnosticcheck/check.go, .github/workflows/release.yml.
package main

import (
	"fmt"
	"github.com/keonho-kim/meca/devtools/internal/diagnosticcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
	"os"
	"strings"
)

func main() {
	if len(os.Args) > 1 && strings.HasPrefix(os.Args[1], "-C=") {
		if err := os.Chdir(strings.TrimPrefix(os.Args[1], "-C=")); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}
	singlechecker.Main(diagnosticcheck.Analyzer)
}

// Purpose: Exercise rejected product diagnostic declarations.
// Pattern: Static analysis fixture.
// Usage: The diagnosticcheck analyzer test loads this package.
// Related: devtools/internal/diagnosticcheck/check_test.go.
package example

import (
	"errors"
	"fmt"
	"log/slog"
)

func invalid() error {
	slog.Info("arbitrary event") // want "declare product errors and logs"
	if true {
		return errors.New("arbitrary error") // want "declare product errors and logs"
	}
	return fmt.Errorf("arbitrary wrapper") // want "declare product errors and logs"
}

func invalidAssignment() {
	value := struct{ Code, Error string }{}
	value.Code = "local_failure"                              // want "declare diagnostic text and codes"
	value.Error = fmt.Sprintf("private failure: %s", "value") // want "declare diagnostic text and codes" "declare diagnostic text and codes"
	value.Error = ""
}

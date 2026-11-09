// Purpose: Keep compatible deadline and joined-cause responses consistent with native classification.
// Pattern: HTTP error contract tests.
// Usage: go test ./internal/serverapi/openai -run TestErrorClassification.
// Related: internal/serverapi/openai/errors.go, internal/diagnostics/fault/classify.go.
package openai

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func TestErrorClassification(t *testing.T) {
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{
		{context.DeadlineExceeded, 504, catalog.DeadlineExceeded},
		{errors.Join(fault.New(catalog.ResourceExhausted, catalog.UnknownFailure), context.Canceled), 429, catalog.ResourceExhausted},
	} {
		w := httptest.NewRecorder()
		(&Handler{}).fail(w, tc.err)
		if w.Code != tc.status || !strings.Contains(w.Body.String(), tc.code) {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

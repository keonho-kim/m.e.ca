// Purpose: Preserve serious joined causes and classify cancellation consistently.
// Pattern: Failure contract regression tests.
// Usage: go test ./internal/diagnostics/fault.
// Related: internal/diagnostics/fault/classify.go, internal/diagnostics/catalog/codes.go.
package fault

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func TestJoinedCauseClassification(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code string
	}{
		{context.Canceled, catalog.Canceled},
		{context.DeadlineExceeded, catalog.DeadlineExceeded},
		{errors.Join(New(catalog.ResourceExhausted, catalog.UnknownFailure), context.Canceled), catalog.ResourceExhausted},
		{errors.Join(context.Canceled, New(catalog.ResourceExhausted, catalog.UnknownFailure)), catalog.ResourceExhausted},
		{errors.Join(context.DeadlineExceeded, New(catalog.PersistenceFailed, catalog.CompletionUncertain)), catalog.PersistenceFailed},
		{fmt.Errorf("wrapper: %w", context.DeadlineExceeded), catalog.DeadlineExceeded},
		{errors.Join(errors.New("unclassified storage failure"), context.Canceled), catalog.OperationFailed},
	} {
		if got := Code(tc.err); got != tc.code {
			t.Fatalf("%v: %s, want %s", tc.err, got, tc.code)
		}
	}
}

type sqliteFailure int

func (e sqliteFailure) Error() string { return "private SQLite detail" }
func (e sqliteFailure) Code() int     { return int(e) }

func TestStorageFailureClassification(t *testing.T) {
	for _, tc := range []struct {
		code int
		want string
	}{{5, catalog.StorageBusy}, {6, catalog.StorageBusy}, {8, catalog.StorageReadOnly}, {13, catalog.StorageFull}, {11, catalog.StorageCorrupt}, {26, catalog.StorageCorrupt}, {10 | 3<<8, catalog.StorageIO}} {
		if got := Code(errors.Join(context.Canceled, sqliteFailure(tc.code))); got != tc.want {
			t.Fatalf("SQLite %d: %s, want %s", tc.code, got, tc.want)
		}
	}
}

func TestUnexaminedCausesCannotBecomeCancellation(t *testing.T) {
	causes := []error{errors.New("serious unexamined failure")}
	for range 256 {
		causes = append(causes, context.Canceled)
	}
	if got := Code(errors.Join(causes...)); got != catalog.OperationFailed {
		t.Fatal("unexamined causes were presented as cancellation", got)
	}
}

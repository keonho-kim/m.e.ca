// Purpose: Select one failure class without hiding serious joined causes behind cancellation.
// Pattern: Bounded error-tree classification.
// Usage: Logs, task settlement and every transport share Code.
// Related: internal/diagnostics/catalog/codes.go, internal/diagnostics/fault/error.go.
package fault

import (
	"context"
	"errors"
	"io/fs"
	"net"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

func Code(err error) string {
	if err == nil {
		return ""
	}
	selected, priority := "", 0
	choose := func(code string) {
		if rank := catalog.Priority(code); rank > priority {
			selected, priority = code, rank
		}
	}
	type branch struct {
		cause      error
		classified bool
	}
	pending := []branch{{cause: err}}
	incomplete := false
	for count := 0; len(pending) > 0 && count < 128; count++ {
		item := pending[len(pending)-1]
		current := item.cause
		pending = pending[:len(pending)-1]
		if current == nil {
			continue
		}
		classified := item.classified
		if driver, ok := current.(interface{ Code() int }); ok {
			code := catalog.SQLiteCode(driver.Code())
			if code != "" {
				choose(code)
				classified = true
			}
		}
		if typed, ok := current.(interface{ ErrorCode() string }); ok {
			code := catalog.Semantic(typed.ErrorCode())
			classified = classified || catalog.Priority(code) > 0
			choose(code)
		}
		if joined, ok := current.(interface{ Unwrap() []error }); ok {
			causes := joined.Unwrap()
			incomplete = incomplete || len(causes) > 128-count
			for _, cause := range causes[:min(len(causes), 128-count)] {
				pending = append(pending, branch{cause, classified})
			}
			continue
		}
		if wrapped, ok := current.(interface{ Unwrap() error }); ok && wrapped.Unwrap() != nil {
			pending = append(pending, branch{wrapped.Unwrap(), classified})
			continue
		}
		switch {
		case errors.Is(current, context.Canceled):
			choose(catalog.Canceled)
		case errors.Is(current, context.DeadlineExceeded):
			choose(catalog.DeadlineExceeded)
		case errors.Is(current, fs.ErrPermission):
			choose(catalog.PermissionDenied)
		case errors.Is(current, fs.ErrNotExist):
			choose(catalog.NotFound)
		default:
			var timeout net.Error
			if errors.As(current, &timeout) && timeout.Timeout() {
				choose(catalog.DeadlineExceeded)
			} else if !classified {
				choose(catalog.OperationFailed)
			}
		}
	}
	if incomplete || len(pending) > 0 {
		choose(catalog.OperationFailed)
	}
	if selected == "" {
		return catalog.OperationFailed
	}
	return selected
}

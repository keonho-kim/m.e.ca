// Purpose: Verify cancellation during initialization and byte-bounded JSON returns.
// Pattern: Controlled cancellation and boundary-value behavior tests.
// Usage: go test ./internal/tools/codemode.
// Related: internal/tools/codemode/runtime.go, internal/tools/codemode/engine.go.
package codemode

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

type startupCancellation struct {
	context.Context
	checks atomic.Int32
	cancel context.CancelFunc
}

func (c *startupCancellation) Err() error {
	if c.checks.Add(1) == 3 {
		c.cancel()
	}
	return c.Context.Err()
}

func TestCancellationDuringRuntimeInstallation(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	ctx := &startupCancellation{Context: parent, cancel: cancel}
	_, err := Run(ctx, `return true;`, nil, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}

func TestReturnedJSONUsesUTF8ByteLimit(t *testing.T) {
	for _, test := range []struct {
		name, code string
		accepted   bool
	}{
		{"ascii boundary", `return "x".repeat(1048574);`, true},
		{"unicode within limit", `return "한".repeat(300000);`, true},
		{"unicode exceeds limit", `return "한".repeat(400000);`, false},
		{"surrogate pair exceeds limit", `return "😀".repeat(300000);`, false},
		{"oversized return", `return "x".repeat(8*1024*1024);`, false},
		{"mutated builtins", `JSON.stringify = () => "0"; encodeURIComponent = () => ""; unescape = () => ""; RegExp.prototype.exec = () => [""]; return "한".repeat(400000);`, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := Run(t.Context(), test.code, nil, nil)
			if (err == nil) != test.accepted {
				t.Fatalf("accepted=%v bytes=%d err=%v", test.accepted, len(result.Value), err)
			}
			if len(result.Value) > MaxResultBytes {
				t.Fatal("oversized result escaped the runtime")
			}
			if err != nil && !strings.Contains(err.Error(), "returned JSON") {
				t.Fatalf("missing actionable size error: %v", err)
			}
		})
	}
}

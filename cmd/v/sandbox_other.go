//go:build !linux

// Purpose: Leave the private Linux execution path outside other platform entry points.
// Pattern: Platform composition boundary.
// Usage: main calls this before ordinary startup on every platform.
// Related: cmd/v/sandbox_linux.go, cmd/v/main.go.
package main

func runSandboxChild() (bool, error) { return false, nil }

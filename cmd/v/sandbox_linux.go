//go:build linux

// Purpose: Dispatch the private sandbox child before bootstrap, storage or terminal initialization.
// Pattern: Composition Root.
// Usage: main checks this entry before ordinary CLI startup.
// Related: internal/tools/local/process/sandbox/linux/child_linux.go, cmd/v/main.go.
package main

import "github.com/keonho-kim/v-code/internal/tools/local/process/sandbox/linux"

func runSandboxChild() (bool, error) { return linux.Child() }

// Purpose: Distinguish invalid shell programs from explicitly blocked command effects.
// Pattern: Typed boundary errors.
// Usage: The local command adapter returns actionable tool failure codes from these causes.
// Related: internal/policy/command/analyze.go, internal/policy/command/effects.go, internal/tools/local/command_safety.go.
package command

import "fmt"

type SyntaxError struct{ Cause error }

func (e *SyntaxError) Error() string { return fmt.Sprintf("invalid shell command: %v", e.Cause) }
func (e *SyntaxError) Unwrap() error { return e.Cause }

type BlockedError struct{ Cause error }

func (e *BlockedError) Error() string { return e.Cause.Error() }
func (e *BlockedError) Unwrap() error { return e.Cause }

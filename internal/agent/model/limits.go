// Purpose: Distinguish incomplete model output from transport failures for safe native call correction.
// Pattern: Typed model boundary error.
// Usage: Provider finish reasons and phase collectors return OutputLimitError.
// Related: internal/provider/diagnostic/check.go, internal/agent/run/generate.go.
package model

type OutputLimitError struct{ Reason string }

func (e *OutputLimitError) Error() string { return "model output truncated: " + e.Reason }

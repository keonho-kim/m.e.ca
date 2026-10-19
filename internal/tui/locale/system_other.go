//go:build !darwin

// Purpose: Use the standard locale environment on platforms without a native fallback reader.
// Pattern: Platform adapter.
// Usage: locale.System selects this implementation outside macOS.
// Related: internal/tui/locale/system.go.
package locale

import "context"

func nativeLanguage(context.Context) string { return "" }

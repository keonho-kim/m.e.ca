// Purpose: Parse the one supported diagnostic level setting for every process mode.
// Pattern: Configuration boundary.
// Usage: cmd/meca applies ParseLevel before startup and reuses it for the mode-specific sink.
// Related: internal/diagnostics/logger.go, internal/diagnostics/catalog/declarations_04.go.
package diagnostics

import (
	"log/slog"
	"strings"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func ParseLevel(value string) (slog.Level, error) {
	switch strings.ToLower(value) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fault.Errorf(catalog.MecaMECALOGLEVELMustBeDebugInfo)
	}
}

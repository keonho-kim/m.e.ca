// Purpose: Carry the process logger and current secret redactor through owned I/O contexts.
// Pattern: Narrow context dependency injection.
// Usage: Composition roots attach Logger; request boundaries attach their Vault redactor.
// Related: internal/diagnostics/logger.go, internal/agent/run/run.go, internal/serverapi/commands.go.
package diagnostics

import (
	"context"
	"log/slog"
)

type loggerKey struct{}
type redactorKey struct{}

func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}
func WithRedactor(ctx context.Context, redact func(string) string) context.Context {
	return context.WithValue(ctx, redactorKey{}, redact)
}
func logger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return nil
	}
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return nil
}

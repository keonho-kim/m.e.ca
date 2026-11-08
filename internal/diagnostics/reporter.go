// Purpose: Expose registered diagnostics without exposing a raw logging backend.
// Pattern: Process-scoped diagnostic boundary.
// Usage: Composition attaches Sink; product packages use Log and WithFields.
// Related: internal/diagnostics/catalog/events.go, internal/diagnostics/context.go.
package diagnostics

import (
	"context"
	"log/slog"
	"sync/atomic"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

var missingReporters atomic.Uint64

func WithSink(ctx context.Context, sink *Sink) context.Context { return withLogger(ctx, sink.logger()) }

func WithSinkContext(ctx, owner context.Context) context.Context {
	if value := logger(owner); value != nil {
		return withLogger(ctx, value)
	}
	return ctx
}

func WithFields(ctx context.Context, fields ...any) context.Context {
	if logger := logger(ctx); logger != nil {
		return withLogger(ctx, logger.With(fields...))
	}
	return ctx
}

func Log(ctx context.Context, event catalog.Event, fields ...any) {
	level, registered := catalog.Level(event)
	if !registered {
		return
	}
	logger := logger(ctx)
	if logger == nil {
		if missingReporters.Add(1) == 1 {
			emergency(catalog.ReporterMissing)
		}
		return
	}
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] == "error_code" {
			if code, ok := fields[i+1].(string); ok {
				switch catalog.Priority(code) {
				case 100, 90, 80, 70:
					level = max(level, slog.LevelError)
				case 60, 50, 30, 20:
					if event == catalog.ControlCompleted || event == catalog.StreamClosed {
						level = slog.LevelDebug
					}
				}
			}
		}
	}
	logger.Log(ctx, level, string(event), fields...)
}

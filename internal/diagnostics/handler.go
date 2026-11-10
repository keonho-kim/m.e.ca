// Purpose: Encode allowlisted diagnostic metadata with bounded strings and context-specific redaction.
// Pattern: Standard slog handler.
// Usage: Sink.Logger creates handlers; request contexts supply the current credential redactor.
// Related: internal/diagnostics/logger.go, internal/diagnostics/context.go.
package diagnostics

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/diagnostics/catalog"
)

type handler struct {
	sink   *Sink
	attrs  []scopedAttr
	groups []string
}

func (h *handler) Enabled(_ context.Context, level slog.Level) bool {
	if h.sink.failed.Load() {
		h.sink.dropped.Add(1)
		return false
	}
	return level >= h.sink.options.Level
}
func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copy := &handler{sink: h.sink, groups: h.groups, attrs: make([]scopedAttr, 0, 32)}
	copy.attrs = append(copy.attrs, h.attrs...)
	for _, attr := range attrs[:min(len(attrs), 32)] {
		copy.attrs = collectAttr(copy.attrs, h.groups, attr)
	}
	return copy
}
func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	copy := &handler{sink: h.sink, attrs: h.attrs, groups: append(append([]string(nil), h.groups...), retainedGroup(name))}
	if len(copy.groups) > 8 {
		copy.groups = copy.groups[:8]
	}
	return copy
}

func allowed(key string) bool {
	switch key {
	case "component", "operation", "request_id", "session_id", "task_id", "run_id", "agent", "call_id", "generation", "attempt", "status", "error_code", "duration_ms", "bytes", "count", "dropped", "skill_id", "skill", "provider", "model", "protocol":
		return true
	case "listen", "persistence", "limit", "workspace_id":
		return true
	}
	return false
}

func safeText(text string, redact func(string) string, limit int) string {
	// Redact a whole bounded value before clipping so credential prefixes cannot leak.
	if len(text) > maxRecordBytes {
		return "[omitted]"
	}
	text = redact(text)
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	if len(text) > limit {
		text = text[:limit]
		for !utf8.ValidString(text) && len(text) > 0 {
			text = text[:len(text)-1]
		}
	}
	return text
}

func (h *handler) Handle(ctx context.Context, record slog.Record) error {
	redact := h.sink.options.Redact
	if current, ok := ctx.Value(redactorKey{}).(func(string) string); ok && current != nil {
		redact = current
	}
	message := safeText(record.Message, redact, 128)
	key := record.Message
	component, code := "", ""
	selectKey := func(attr slog.Attr) {
		if attr.Value.Kind() != slog.KindString {
			return
		}
		if attr.Key == "component" {
			component = attr.Value.String()
		}
		if attr.Key == "error_code" {
			code = attr.Value.String()
		}
	}
	for _, scoped := range h.attrs {
		if len(scoped.groups) == 0 {
			selectKey(scoped.attr)
		}
	}
	record.Attrs(func(attr slog.Attr) bool { selectKey(attr); return true })
	key += "|" + component + "|" + code
	if !h.sink.accept(record.Level, key) {
		return nil
	}
	at := record.Time
	if at.IsZero() {
		at = h.sink.options.Now()
	}
	fields := map[string]any{"time": at.UTC().Format(time.RFC3339Nano), "level": record.Level.String(), "msg": message}
	attrs := append([]scopedAttr(nil), h.attrs...)
	visited := 0
	record.Attrs(func(attr slog.Attr) bool { visited++; attrs = collectAttr(attrs, h.groups, attr); return visited < 32 })
	for _, scoped := range attrs {
		attr, value := scoped.attr, scoped.attr.Value
		target := fields
		valid := true
		for _, group := range scoped.groups {
			name := safeText(group, redact, 64)
			child, ok := target[name].(map[string]any)
			if !ok {
				if target[name] != nil {
					valid = false
					break
				}
				child = make(map[string]any)
				target[name] = child
			}
			target = child
		}
		if !valid {
			continue
		}
		switch value.Kind() {
		case slog.KindString:
			target[attr.Key] = safeText(value.String(), redact, 256)
		case slog.KindInt64:
			target[attr.Key] = value.Int64()
		case slog.KindUint64:
			target[attr.Key] = value.Uint64()
		case slog.KindBool:
			target[attr.Key] = value.Bool()
		case slog.KindDuration:
			target[attr.Key] = value.Duration().Milliseconds()
		}
	}
	body, err := json.Marshal(fields)
	if err != nil {
		return err
	}
	if len(body) >= maxRecordBytes {
		body, _ = json.Marshal(map[string]any{"time": at.UTC().Format(time.RFC3339Nano), "level": record.Level.String(), "msg": message, "error_code": catalog.DiagnosticRecordLimitCode})
	}
	h.sink.enqueue(record.Level, append(body, '\n'))
	return nil
}

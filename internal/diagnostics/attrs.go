// Purpose: Bound retained slog attributes while preserving their declared group scope.
// Pattern: Pure diagnostic metadata selection.
// Usage: Handler.WithAttrs and Handle share this allowlisted projection.
// Related: internal/diagnostics/handler.go.
package diagnostics

import (
	"log/slog"
	"strings"
)

type scopedAttr struct {
	groups []string
	attr   slog.Attr
}

func retainedGroup(name string) string {
	if len(name) > 64 {
		return "[omitted]"
	}
	return strings.Clone(name)
}

func collectAttr(out []scopedAttr, groups []string, attr slog.Attr) []scopedAttr {
	if len(out) >= 32 || len(groups) > 8 {
		return out
	}
	value := attr.Value.Resolve()
	if value.Kind() == slog.KindGroup {
		if attr.Key != "" {
			groups = append(append([]string(nil), groups...), retainedGroup(attr.Key))
		}
		children := value.Group()
		for _, child := range children[:min(len(children), 32)] {
			out = collectAttr(out, groups, child)
		}
		return out
	}
	if !allowed(attr.Key) {
		return out
	}
	attr.Key = strings.Clone(attr.Key)
	switch value.Kind() {
	case slog.KindString:
		text := value.String()
		if len(text) > maxRecordBytes {
			text = "[omitted]"
		}
		attr.Value = slog.StringValue(strings.Clone(text))
	case slog.KindInt64, slog.KindUint64, slog.KindBool, slog.KindDuration:
		attr.Value = value
	default:
		return out
	}
	return append(out, scopedAttr{groups: groups, attr: attr})
}

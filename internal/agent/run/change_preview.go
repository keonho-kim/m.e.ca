// Purpose: Bound persisted applied-change previews without expanding model context or hiding truncation.
// Pattern: Pure presentation payload preparation.
// Usage: Successful file execution copies its frozen preview into the final event.
// Related: internal/agent/run/execute.go, internal/agent/record/event_validation.go.
package run

import (
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"unicode/utf8"
)

func boundedChangePreview(text string) string {
	if len(text) <= runrecord.MaxChangePreviewBytes {
		return text
	}
	const notice = "\n[변경 내용 일부 생략: 표시 한도 16MiB]\n"
	end := runrecord.MaxChangePreviewBytes - len(notice)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + notice
}

// Purpose: Bound persisted applied-change previews without expanding model context or hiding truncation.
// Pattern: Pure presentation payload preparation.
// Usage: Successful file execution copies its frozen preview into the final event.
// Related: internal/agent/run/execute.go, internal/agent/run/event_validation.go.
package run

import "unicode/utf8"

const maxChangePreviewBytes = 128 * 1024

func boundedChangePreview(text string) string {
	if len(text) <= maxChangePreviewBytes {
		return text
	}
	const notice = "\n[변경 내용 일부 생략: 표시 한도 128KiB]\n"
	end := maxChangePreviewBytes - len(notice)
	for end > 0 && !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + notice
}

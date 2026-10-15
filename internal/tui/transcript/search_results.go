// Purpose: Render file searches, content matches and directory entries as compact lists.
// Pattern: Pure presentation projection.
// Usage: Tool result dispatch passes stored JSON; file paths and source text remain literal.
// Related: internal/tui/transcript/details.go, internal/tools/local/search/search.go.
package transcript

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func (f Formatter) searchOutput(body []byte, name string) string {
	var result struct{ Matches json.RawMessage }
	if json.Unmarshal(body, &result) != nil || len(result.Matches) == 0 {
		return ""
	}
	var matches []struct {
		Path string
		Line int
		Text string
	}
	if json.Unmarshal(result.Matches, &matches) != nil {
		return ""
	}
	if len(matches) == 0 {
		return indentTool("검색 결과 없음")
	}
	var text strings.Builder
	if name == "search_files" {
		text.WriteString(indentTool(fmt.Sprintf("파일 %d개", len(matches))))
		for _, match := range matches {
			text.WriteString(indentTool(f.Redact(match.Path)))
		}
		return text.String()
	}
	text.WriteString(indentTool(fmt.Sprintf("일치 %d건", len(matches))))
	width := 3
	for _, match := range matches {
		width = max(width, len(strconv.Itoa(match.Line)))
	}
	previous := ""
	for _, match := range matches {
		if match.Path != previous {
			text.WriteString(indentTool(f.Redact(match.Path)))
			previous = match.Path
		}
		number := "?"
		if match.Line > 0 {
			number = strconv.Itoa(match.Line)
		}
		text.WriteString(indentTool(fmt.Sprintf("  %*s | %s", width, number, f.Redact(match.Text))))
	}
	return text.String()
}

func (f Formatter) directoryOutput(body []byte) string {
	var result struct{ Entries json.RawMessage }
	if json.Unmarshal(body, &result) != nil || len(result.Entries) == 0 {
		return ""
	}
	var entries []struct {
		Name      string
		Directory bool
	}
	if json.Unmarshal(result.Entries, &entries) != nil {
		return ""
	}
	if len(entries) == 0 {
		return indentTool("항목 없음")
	}
	var text strings.Builder
	text.WriteString(indentTool(fmt.Sprintf("항목 %d개", len(entries))))
	for _, entry := range entries {
		name := entry.Name
		if entry.Directory {
			name += "/"
		}
		text.WriteString(indentTool(f.Redact(name)))
	}
	return text.String()
}

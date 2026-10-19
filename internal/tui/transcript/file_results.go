// Purpose: Describe applied file operations, line ranges and actual changed content.
// Pattern: Pure presentation projection.
// Usage: Tool result dispatch supplies the event and immutable result bytes.
// Related: internal/tui/transcript/details.go, internal/agent/tool/execution.go.
package transcript

import (
	"encoding/json"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	"github.com/keonho-kim/v-code/internal/agent/tool"
)

func lineRange(first, last int) string {
	if first == last || last < first {
		return fmt.Sprint(first)
	}
	return fmt.Sprintf("%d–%d", first, last)
}

func (f Formatter) fileOutput(event run.Event, body []byte) string {
	var result struct {
		Created []string
		Changed *bool
		Lines   []tool.LineChange
	}
	if json.Unmarshal(body, &result) != nil || result.Changed == nil {
		return ""
	}
	if event.Call.Name == "create_directory" {
		if len(result.Created) == 0 {
			if event.Kind != "tool_succeeded" {
				return ""
			}
			return indentTool(f.Language.Text("요청한 폴더가 이미 존재합니다"))
		}
		text := indentTool(f.Language.Format("폴더 생성 완료 · %d개 (상위 폴더 포함)", len(result.Created)))
		if event.Kind != "tool_succeeded" {
			text = f.resultSummary(event) + indentTool(f.Language.Format("중단 전 생성한 폴더 · %d개", len(result.Created)))
		}
		for _, path := range result.Created {
			text += indentTool(f.Redact(path))
		}
		return text
	}
	if event.Kind != "tool_succeeded" {
		return ""
	}
	if !*result.Changed {
		return indentTool(f.Language.Text("변경 없음"))
	}
	label := map[string]string{"write_file": "파일 저장 완료", "edit_file": "수정 완료", "create_directory": "폴더 생성 완료", "move_file": "이동 완료"}[event.Call.Name]
	if event.Operation != nil {
		for _, change := range event.Operation.Changes {
			if change.BeforeDigest != "" && change.BeforeDigest == change.AfterDigest {
				label = "변경 없음"
			}
			if event.Call.Name == "write_file" && change.BeforeDigest == "missing" {
				label = "파일 생성 완료"
			}
		}
	}
	text := indentTool(f.Language.Text(label))
	for _, lines := range result.Lines {
		if lines.OldStartLine > 0 {
			old := lineRange(lines.OldStartLine, lines.OldEndLine)
			if lines.Removed {
				text += indentTool(f.Language.Text("삭제한 줄: ") + old)
			} else if lines.NewStartLine > 0 {
				text += indentTool(f.Language.Format("기존 %s줄 → %s줄", old, lineRange(lines.NewStartLine, lines.NewEndLine)))
			}
		}
	}
	if event.ChangePreview != "" {
		text += "\n" + indentTool(f.Language.Text("Changes")) + renderChangePreview(f.Redact(event.ChangePreview), f.Language)
	}
	return text
}

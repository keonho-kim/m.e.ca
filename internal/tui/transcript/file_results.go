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
		Changed *bool
		Lines   *tool.LineChange
	}
	if json.Unmarshal(body, &result) != nil || result.Changed == nil || event.Kind != "tool_succeeded" {
		return ""
	}
	if !*result.Changed {
		return indentTool("변경 없음")
	}
	label := map[string]string{"write_file": "파일 저장 완료", "edit_file": "수정 완료", "delete_file": "삭제 완료", "create_directory": "폴더 생성 완료", "move_file": "이동 완료"}[event.Call.Name]
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
	text := indentTool(label)
	if lines := result.Lines; lines != nil && lines.OldStartLine > 0 {
		old := lineRange(lines.OldStartLine, lines.OldEndLine)
		if lines.Removed {
			text += indentTool("삭제한 줄: " + old)
		} else if lines.NewStartLine > 0 {
			text += indentTool("기존 " + old + "줄 → " + lineRange(lines.NewStartLine, lines.NewEndLine) + "줄")
		}
	}
	if event.ChangePreview != "" {
		text += "\n" + indentTool("변경 내용") + renderChangePreview(f.Redact(event.ChangePreview))
	}
	return text
}

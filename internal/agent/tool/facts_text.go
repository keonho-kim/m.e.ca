// Purpose: Describe actual partial effects when a model cannot provide its final report.
// Pattern: Deterministic report projection.
// Usage: Run finalization and recovery render Facts without another model call.
// Related: internal/agent/tool/facts.go, internal/agent/run/report.go.
package tool

import (
	"fmt"
	"strings"
)

func (f Facts) Describe() string {
	var text strings.Builder
	shown, total := 0, 0
	for _, file := range f.Files {
		if !file.Changed {
			continue
		}
		total++
		if shown >= 20 {
			continue
		}
		shown++
		label := "기록된 변경"
		if file.Stale {
			label += "(현재 상태 재확인 필요)"
		}
		fmt.Fprintf(&text, "- %s: %s\n", label, file.Path)
	}
	if total > shown {
		fmt.Fprintf(&text, "- 추가 변경 %d개: 보고의 files에 경로와 버전을 보존했습니다.\n", total-shown)
	}
	start := max(0, len(f.Commands)-6)
	if start > 0 {
		fmt.Fprintf(&text, "- 이전 명령 %d개는 실행 기록에 보존했습니다.\n", start)
	}
	for _, command := range f.Commands[start:] {
		fmt.Fprintf(&text, "- 실행 기록: %s · %s", command.Command, command.Status)
		if command.ExitCode != nil {
			fmt.Fprintf(&text, " · exit %d", *command.ExitCode)
		}
		text.WriteByte('\n')
	}
	if total == 0 && len(f.Commands) == 0 {
		text.WriteString("- 확인된 파일 변경·명령 실행 기록이 없습니다.\n")
	}
	return text.String()
}

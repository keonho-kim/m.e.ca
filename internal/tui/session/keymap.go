// Purpose: Choose keyboard help from the current screen and focus and reserve fixed footer rows.
// Pattern: Presentation projection.
// Usage: Rendering and scrolling share the same content-height calculation.
// Related: internal/tui/session/render.go, internal/tui/session/keys.go, internal/tui/view/keymap.go.
package session

import (
	planning "github.com/keonho-kim/v-code/internal/agent/plan"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) shortcuts() []view.Shortcut {
	shortcuts := []view.Shortcut{{Key: "^C", Label: "종료"}}
	var controls []view.Shortcut
	switch a.page {
	case "chat":
		switch {
		case a.commandOpen:
			controls = []view.Shortcut{
				{Key: "↑↓", Label: "명령 선택"},
				{Key: "Enter", Label: "실행"},
				{Key: "Tab", Label: "삽입"},
				{Key: "Esc", Label: "닫기"},
			}
		case a.refOpen:
			controls = []view.Shortcut{
				{Key: "↑↓", Label: "참조 선택"},
				{Key: "Enter", Label: "삽입"},
				{Key: "Esc", Label: "닫기"},
			}
		default:
			controls = []view.Shortcut{
				{Key: "Enter", Label: "전송"},
				{Key: "!new", Label: "새 세션"},
				{Key: "Alt+Enter", Label: "줄바꿈"},
				{Key: "↑↓", Label: "기록"},
				{Key: "PgUp/Dn", Label: "스크롤"},
				{Key: "!", Label: "명령"},
				{Key: "@", Label: "참조"},
			}
			if a.busy {
				controls = controls[1:]
			} else if a.planMode && a.plan != nil && a.plan.Status == planning.AwaitingApproval && a.editor.String() == "" {
				controls[0].Label = "승인 메뉴"
			}
		}
	case "files":
		switch {
		case !a.browser.Input && fileCommand(a.menuInput.String()):
			controls = []view.Shortcut{
				{Key: "Enter", Label: "실행"},
				{Key: "Esc", Label: "취소"},
				{Key: "Tab", Label: "입력"},
			}
		case a.browser.Input:
			controls = []view.Shortcut{
				{Key: "Enter", Label: "전송"},
				{Key: "Tab", Label: "탐색"},
				{Key: ":q", Label: "돌아가기"},
			}
		default:
			controls = []view.Shortcut{
				{Key: "↑↓", Label: "이동"},
				{Key: "←→", Label: "열 이동"},
				{Key: "Enter", Label: "폴더 열기"},
				{Key: "Space", Label: "선택"},
				{Key: "Tab", Label: "입력"},
				{Key: ":q", Label: "돌아가기"},
			}
		}
	case "endpoint", "key":
		controls = []view.Shortcut{
			{Key: "Enter", Label: "다음"},
			{Key: "^U", Label: "전체 삭제"},
			{Key: "←→", Label: "커서"},
			{Key: "Esc", Label: "돌아가기"},
		}
	case "manual":
		controls = []view.Shortcut{{Key: "Enter", Label: "연결"}, {Key: "↑↓", Label: "후보 선택"}, {Key: "^U", Label: "전체 삭제"}, {Key: "Esc", Label: "돌아가기"}}
		if a.modelHintsOpen && len(a.modelHints) > 0 {
			controls[0].Label = "모델명 입력"
			controls = append(controls, view.Shortcut{Key: "Tab", Label: "모델명 입력"})
		}
		if a.modelHintsOpen {
			controls[3].Label = "목록 닫기"
		}
	case "effort":
		controls = []view.Shortcut{{Key: "↑↓", Label: "강도 선택"}, {Key: "Enter", Label: "검증 후 적용"}, {Key: "Esc", Label: "돌아가기"}}
	case "modelinfo":
		controls = []view.Shortcut{{Key: "PgUp/Dn", Label: "정보 스크롤"}, {Key: "Enter", Label: "모델 메뉴"}, {Key: "Esc", Label: "모델 메뉴"}}
	case "approval":
		controls = []view.Shortcut{
			{Key: "↑↓", Label: "선택"},
			{Key: "Enter", Label: "선택 실행"},
			{Key: "PgUp/Dn", Label: "diff"},
			{Key: "Esc", Label: "거부"},
		}
	default:
		controls = []view.Shortcut{
			{Key: "↑↓", Label: "선택"},
			{Key: "Enter", Label: "선택 실행"},
			{Key: "Esc", Label: "돌아가기"},
		}
		if a.page == "model" && a.choice == 1 && a.selection.ThinkingSupported {
			controls = append(controls, view.Shortcut{Key: "←→", Label: "Thinking"})
		}
	}
	if a.agentActive && a.busy {
		shortcuts = append(shortcuts, view.Shortcut{Key: "Esc", Label: "작업 중지"})
		if a.page == "chat" || a.page == "files" && a.browser.Input {
			shortcuts = append(shortcuts, view.Shortcut{Key: "Tab", Label: "다음 요청 예약"})
		}
		filtered := controls[:0]
		for _, control := range controls {
			if control.Key != "Esc" && !(control.Key == "Tab" && (a.page == "chat" || a.page == "files" && a.browser.Input)) {
				filtered = append(filtered, control)
			}
		}
		controls = filtered
	}
	return append(shortcuts, controls...)
}

func (a *app) bodyHeight(height int) int {
	return max(0, height-view.FooterRows)
}

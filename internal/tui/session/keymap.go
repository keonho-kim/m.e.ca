// Purpose: Select keyboard hints from actual focus, available targets and worker state.
// Pattern: Presentation projection.
// Usage: Rendering reserves the footer and projects only currently useful key actions.
// Related: internal/tui/session/keys.go, internal/tui/session/files.go, internal/tui/session/input_region.go, internal/tui/view/keymap.go.
package session

import (
	"github.com/keonho-kim/v-code/internal/commands"
	"github.com/keonho-kim/v-code/internal/tui/view"
)

func (a *app) shortcuts() []view.Shortcut {
	if a.textSelection.Active() {
		keys := []view.Shortcut{{Key: "^C / Cmd+C", Label: "복사"}, {Key: "Esc", Label: "선택 해제"}}
		if _, _, ok := a.selectedInputRange(); ok {
			keys = append(keys, view.Shortcut{Key: "Backspace/Del", Label: "선택 삭제"})
		}
		return keys
	}
	shortcuts := []view.Shortcut{{Key: "^C", Label: "종료"}}
	if a.filePreview != nil {
		controls := []view.Shortcut{{Key: "Esc", Label: "닫기"}}
		if fileCommand(a.menuInput.String()) {
			controls = append(controls, view.Shortcut{Key: "Enter", Label: a.fileCommandLabel()}, view.Shortcut{Key: "^U", Label: "지우기"})
			return append(shortcuts, controls...)
		}
		if a.filePreview.Full {
			controls = append(controls, view.Shortcut{Key: "Tab", Label: "반쪽"})
		} else {
			controls = append(controls, view.Shortcut{Key: "Enter", Label: "전체화면"})
		}
		controls = append(controls, view.Shortcut{Key: "↑↓", Label: "스크롤"}, view.Shortcut{Key: "PgUp/Dn", Label: "페이지"}, view.Shortcut{Key: "Home/End", Label: "처음/끝"})
		return append(shortcuts, controls...)
	}
	if a.directoryLoading && (a.page != "files" || a.browser.Input || !fileCommand(a.menuInput.String())) {
		return append(shortcuts, view.Shortcut{Key: "Esc", Label: "조회 취소"}, view.Shortcut{Key: ":q", Label: "복귀"})
	}
	var controls []view.Shortcut
	switch a.page {
	case "chat":
		if a.commandOpen {
			label := "실행"
			if len(a.commandHints) > 0 {
				command, err := commands.Parse(a.commandHints[a.commandIndex].Name)
				if err == nil && !a.canSubmit(command.Kind) {
					label = "입력만"
				}
			}
			controls = []view.Shortcut{{Key: "Esc", Label: "닫기"}, {Key: "Shift+Enter", Label: "줄바꿈"}}
			if len(a.commandHints) > 0 {
				controls = append(controls, view.Shortcut{Key: "Enter", Label: label}, view.Shortcut{Key: "↑↓", Label: "선택"}, view.Shortcut{Key: "Tab", Label: "삽입"})
			}
		} else {
			controls = a.inputShortcuts()
			controls = append(controls, view.Shortcut{Key: "PgUp/Dn", Label: "스크롤"}, view.Shortcut{Key: "!", Label: "명령"}, view.Shortcut{Key: "^O", Label: "출력 접기/펴기"})
		}
	case "files":
		if !a.browser.Input && fileCommand(a.menuInput.String()) {
			controls = []view.Shortcut{{Key: "Enter", Label: a.fileCommandLabel()}, {Key: "Esc", Label: "취소"}, {Key: "Shift+Tab", Label: "입력"}, {Key: "^U", Label: "지우기"}}
		} else if a.browser.Input {
			controls = a.inputShortcuts()
			controls = append(controls, view.Shortcut{Key: "Shift+Tab", Label: "탐색"}, view.Shortcut{Key: ":q", Label: "복귀"})
		} else {
			controls = []view.Shortcut{{Key: "Shift+Tab", Label: "입력"}, {Key: ":q", Label: "복귀"}}
			path, directory := a.browser.Target()
			if path != "" {
				controls = append(controls, view.Shortcut{Key: "↑↓", Label: "이동"}, view.Shortcut{Key: "Space", Label: "선택"})
				if directory {
					controls = append(controls, view.Shortcut{Key: "→", Label: "열기"})
				} else {
					controls = append(controls, view.Shortcut{Key: "Tab", Label: "반쪽"}, view.Shortcut{Key: "Enter", Label: "전체화면"})
				}
			}
			if a.browser.Active > 0 {
				controls = append(controls, view.Shortcut{Key: "←", Label: "상위"})
			}
			if c := a.browser.Current(); c != nil && c.Offset > 0 {
				controls = append(controls, view.Shortcut{Key: "PgUp", Label: "이전 목록"})
			}
		}
	case "endpoint", "key":
		label := "다음"
		if a.page == "key" {
			label = "모델 조회"
		}
		controls = []view.Shortcut{{Key: "Enter", Label: label}, {Key: "^U", Label: "지우기"}, {Key: "←→", Label: "커서"}, {Key: "Esc", Label: "취소"}}
	case "manual":
		controls = []view.Shortcut{{Key: "Enter", Label: "연결"}, {Key: "↑↓", Label: "후보 열기"}, {Key: "^U", Label: "지우기"}, {Key: "Esc", Label: "취소"}}
		if a.modelHintsOpen {
			controls[3].Label = "목록 닫기"
			if len(a.modelHints) > 0 {
				controls[0].Label = "입력"
				controls[1].Label = "선택"
				controls = append(controls, view.Shortcut{Key: "Tab", Label: "입력"})
			} else {
				controls = append(controls[:1], controls[2:]...)
			}
		}
	case "effort":
		controls = []view.Shortcut{{Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "적용"}, {Key: "Esc", Label: "닫기"}}
	case "modelinfo":
		controls = []view.Shortcut{{Key: "PgUp/Dn", Label: "스크롤"}, {Key: "Enter", Label: "모델 메뉴"}, {Key: "Esc", Label: "모델 메뉴"}}
	case "approval":
		controls = []view.Shortcut{{Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "확정"}, {Key: "PgUp/Dn", Label: "diff"}, {Key: "Esc", Label: "거부"}}
	default:
		controls = []view.Shortcut{{Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "선택 실행"}, {Key: "Esc", Label: "닫기"}}
		if a.page == "model" && a.choice == 1 && a.selection.ThinkingSupported {
			controls = append(controls, view.Shortcut{Key: "←→", Label: "Thinking"})
		}
	}
	if a.page == "endpoint" || a.page == "key" || a.page == "manual" {
		command, err := commands.Parse(a.menuInput.String())
		if err == nil && (command.Kind == commands.Quit || command.Kind == commands.Exit || command.Kind == commands.NewSession) {
			escape := "취소"
			if a.page == "manual" && a.modelHintsOpen {
				escape = "목록 닫기"
			}
			controls = []view.Shortcut{{Key: "Enter", Label: a.commandEnterLabel(command.Kind)}, {Key: "^U", Label: "지우기"}, {Key: "←→", Label: "커서"}, {Key: "Esc", Label: escape}}
		}
	}
	if a.busy && !a.stopRequested && (a.page == "chat" && !a.commandOpen || a.page == "files" && !fileCommand(a.menuInput.String())) {
		label := "중지"
		if !a.agentActive {
			label = "연결 취소"
		}
		shortcuts = append(shortcuts, view.Shortcut{Key: "Esc", Label: label})
		filtered := controls[:0]
		for _, c := range controls {
			if c.Key != "Esc" {
				filtered = append(filtered, c)
			}
		}
		controls = filtered
	}
	return append(shortcuts, controls...)
}

func (a *app) inputShortcuts() []view.Shortcut {
	controls := []view.Shortcut{{Key: "Shift+Enter", Label: "줄바꿈"}, {Key: "↑↓", Label: "줄 이동"}, {Key: "Ctrl/Cmd+↑↓", Label: "기록"}}
	label := "전송"
	command, err := commands.Parse(a.editor.String())
	if err != nil {
		return controls
	}
	if command.Kind == commands.Quit {
		return append(controls, view.Shortcut{Key: "Enter", Label: a.commandEnterLabel(command.Kind)})
	}
	if a.agentActive && a.busy && !a.stopRequested && (command.Kind == commands.Message || command.Kind == commands.Empty) {
		controls = append(controls, view.Shortcut{Key: "Enter", Label: "방향 수정"})
		if a.pending == "" {
			controls = append(controls, view.Shortcut{Key: "Tab", Label: "후속 예약"})
		}
		return controls
	}
	if a.busy && !a.canSubmit(command.Kind) {
		return controls
	}
	if command.Kind != commands.Message && command.Kind != commands.Empty {
		label = a.commandEnterLabel(command.Kind)
	}
	return append(controls, view.Shortcut{Key: "Enter", Label: label})
}

func (a *app) commandEnterLabel(kind commands.Kind) string {
	switch kind {
	case commands.Quit:
		if a.filePreview != nil {
			return "닫기"
		}
		if a.page != "chat" {
			return "복귀"
		}
		if a.busy {
			return "종료 확인"
		}
		return "종료"
	case commands.Exit:
		return "종료"
	case commands.NewSession:
		return "새 대화"
	default:
		return "실행"
	}
}

func (a *app) fileCommandLabel() string {
	command, err := commands.Parse(a.menuInput.String())
	if err == nil && (command.Kind == commands.Quit || command.Kind == commands.Exit || command.Kind == commands.NewSession) {
		return a.commandEnterLabel(command.Kind)
	}
	return "확인"
}

func (a *app) bodyHeight(height int) int { return max(0, height-view.FooterRows) }

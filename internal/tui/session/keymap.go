// Purpose: Select keyboard hints from actual focus, available targets and worker state.
// Pattern: Presentation projection.
// Usage: Rendering reserves the footer and projects only currently useful key actions.
// Related: internal/tui/session/keys.go, internal/tui/session/files.go, internal/tui/session/input_region.go, internal/tui/view/keymap.go.
package session

import (
	"github.com/keonho-kim/meca/internal/commands"
	"github.com/keonho-kim/meca/internal/tui/view"
)

func (a *app) shortcuts() []view.Shortcut {
	keys := a.screenShortcuts()
	for i := range keys {
		keys[i].Label = a.language.Text(keys[i].Label)
	}
	return keys
}

func (a *app) screenShortcuts() []view.Shortcut {
	if a.textSelection.Active() {
		keys := []view.Shortcut{{Key: "^C / Cmd+C", Label: "복사"}, {Key: "Esc", Label: "선택 해제"}}
		if _, _, ok := a.selectedInputRange(); ok {
			keys = append(keys, view.Shortcut{Key: "Backspace/Del", Label: "선택 삭제"})
		}
		return keys
	}
	if a.page == "model" {
		return a.modelSettingsShortcuts()
	}
	if a.page == "language" {
		return a.languageShortcuts()
	}
	if a.page == "permission" {
		return a.permissionShortcuts()
	}
	if a.page == "chat" && a.queued.Edit != nil {
		return a.pendingEditShortcuts()
	}
	shortcuts := []view.Shortcut{{Key: "^C", Label: "종료"}}
	if a.page == "files" && a.fileSendConfirm {
		_, label := a.fileInputAction()
		return append(shortcuts, view.Shortcut{Key: "y", Label: label}, view.Shortcut{Key: "N", Label: "취소"}, view.Shortcut{Key: "Esc", Label: "Keep writing"})
	}
	if a.page == "files" && a.filePreview != nil {
		controls := []view.Shortcut{{Key: "Esc", Label: "닫기"}}
		if a.filePreview.Full {
			controls = append(controls, view.Shortcut{Key: "Tab", Label: "분할 보기"})
		} else {
			controls = append(controls, view.Shortcut{Key: "Enter", Label: "전체 화면"})
		}
		if a.filePreview.IsMarkdown() {
			label := "Show raw"
			if a.filePreview.Raw {
				label = "Render"
			}
			controls = append(controls, view.Shortcut{Key: "Shift+R", Label: label})
		}
		controls = append(controls, view.Shortcut{Key: "↑↓", Label: "스크롤"}, view.Shortcut{Key: "PgUp/Dn", Label: "페이지"}, view.Shortcut{Key: "Home/End", Label: "처음/끝"})
		return append(shortcuts, controls...)
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
			controls = append(controls, view.Shortcut{Key: "PgUp/Dn", Label: "스크롤"}, view.Shortcut{Key: "!", Label: "명령"})
		}
	case "files":
		if a.browser.Input {
			controls = a.inputShortcuts()
			controls = append(controls, view.Shortcut{Key: "Esc", Label: "닫기"}, view.Shortcut{Key: "Shift+Tab", Label: "탐색"})
		} else {
			controls = []view.Shortcut{{Key: "Esc", Label: "닫기"}, {Key: "Shift+Tab", Label: "입력"}}
			if a.directoryLoading {
				break
			}
			path, directory := a.browser.Target()
			if path != "" {
				controls = append(controls, view.Shortcut{Key: "↑↓", Label: "이동"}, view.Shortcut{Key: "Space", Label: "선택 표시"})
				if directory {
					controls = append(controls, view.Shortcut{Key: "→", Label: "열기"})
				} else {
					controls = append(controls, view.Shortcut{Key: "Tab", Label: "분할 보기"}, view.Shortcut{Key: "Enter", Label: "전체 화면"})
				}
			}
			if a.browser.Active > 0 {
				controls = append(controls, view.Shortcut{Key: "←", Label: "상위"})
			}
			if c := a.browser.Current(); c != nil && c.Offset > 0 {
				controls = append(controls, view.Shortcut{Key: "PgUp", Label: "이전 목록"})
			}
		}
	case "approval":
		controls = []view.Shortcut{{Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "확정"}, {Key: "PgUp/Dn", Label: "변경 내용"}, {Key: "Esc", Label: "거부"}}
	default:
		controls = []view.Shortcut{{Key: "↑↓", Label: "선택"}, {Key: "Enter", Label: "확정"}, {Key: "Esc", Label: "닫기"}}
	}
	if a.page == "chat" && !a.commandOpen && a.fileSubmission != nil {
		shortcuts = append(shortcuts, view.Shortcut{Key: "Esc", Label: "Cancel send"})
	} else if a.worker.active() && !a.worker.stopping() && a.page == "chat" && !a.commandOpen {
		label := "중지"
		if !a.worker.agent() {
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
	if a.output.HasDetails() && (a.page == "chat" && !a.commandOpen || a.page == "files" && a.browser.Input) {
		label := "도구 접기"
		if a.output.Collapsed {
			label = "도구 펴기"
		}
		controls = append(controls, view.Shortcut{Key: "^O", Label: label})
	}
	if a.canEditPending() {
		controls = append(controls, view.Shortcut{Key: "Shift+↑/←", Label: "Edit queue"})
	}
	return append(shortcuts, controls...)
}

func (a *app) inputShortcuts() []view.Shortcut {
	controls := []view.Shortcut{{Key: "Shift+Enter", Label: "줄바꿈"}, {Key: "Ctrl/Cmd+↑↓", Label: "입력 기록"}}
	label := "전송"
	command, err := a.inputCommand()
	if err != nil {
		return controls
	}
	if command.Kind == commands.Quit {
		return append(controls, view.Shortcut{Key: "Enter", Label: a.commandEnterLabel(command.Kind)})
	}
	if a.page == "files" && (command.Kind == commands.Message || command.Kind == commands.Empty) {
		label = "Review send"
	}
	if a.worker.agent() && a.worker.active() && !a.worker.stopping() && (command.Kind == commands.Message || command.Kind == commands.Empty) {
		if a.page == "files" {
			label = "Review steer"
		} else {
			label = "방향 수정"
		}
		controls = append(controls, view.Shortcut{Key: "Enter", Label: label})
		label := "후속 예약"
		if a.page == "files" {
			label = "Review queue"
		}
		controls = append(controls, view.Shortcut{Key: "Tab", Label: label})
		return controls
	}
	if a.worker.active() && !a.canSubmit(command.Kind) {
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
		if a.worker.active() {
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

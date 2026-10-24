// Purpose: Route execution approval and exit confirmation choices.
// Pattern: Presentation command routing.
// Usage: Generic menu keys call choose after the highlighted choice is confirmed.
// Related: internal/tui/session/menu_keys.go, internal/tui/session/commands.go.
package session

func (a *app) choose() {
	switch a.page {
	case "quit":
		if a.choice == 1 {
			a.requestExit()
		} else {
			a.page = "chat"
		}
	case "approval":
		a.decide(a.choice == 0)
	}
}

// Purpose: Build request fixtures and inspect their text without hiding captured path ownership.
// Pattern: Test fixture functions.
// Usage: Queue interaction tests supply ordinary messages and compare visible text.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go.
package session

import requestinput "github.com/keonho-kim/meca/internal/commands/input"

func queuedRequests(texts ...string) []requestinput.Request {
	out := make([]requestinput.Request, len(texts))
	for i, text := range texts {
		out[i].Text = text
	}
	return out
}

func queuedTexts(requests []requestinput.Request) []string {
	out := make([]string, len(requests))
	for i, request := range requests {
		out[i] = request.Text
	}
	return out
}

// Purpose: Build request fixtures and inspect their text without hiding captured path ownership.
// Pattern: Test fixture functions.
// Usage: Queue interaction tests supply ordinary messages and compare visible text.
// Related: internal/tui/composer/request.go, internal/tui/composer/queue/state.go.
package session

import "github.com/keonho-kim/meca/internal/tui/composer"

func queuedRequests(texts ...string) []composer.Request {
	out := make([]composer.Request, len(texts))
	for i, text := range texts {
		out[i].Text = text
	}
	return out
}

func queuedTexts(requests []composer.Request) []string {
	out := make([]string, len(requests))
	for i, request := range requests {
		out[i] = request.Text
	}
	return out
}

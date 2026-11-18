// Purpose: Retire delivered browser text from live model history while preserving the native tool response and unread cursor.
// Pattern: Execution-owned acknowledgement at a completed model decision.
// Usage: Context loading replays durable acknowledgements and drive advances the boundary on success.
// Related: internal/agent/run/loop.go, internal/agent/run/context_memory.go, internal/tools/browser/paging.go.
package run

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
)

func browserTextPage(m modelapi.Message) bool {
	return m.Role == "tool" && m.ToolName == "browser_inspect" && len(m.Images) == 0
}

func (s *execution) acknowledgeBrowserPages(through int) {
	for i := s.browserReadThrough; i < through; i++ {
		m := &s.messages[i]
		if !browserTextPage(*m) {
			continue
		}
		receipt, ok := toolapi.ReadBrowserPage(m.Content)
		if !ok {
			continue
		}
		m.Content = receipt
		s.projection.ForgetMessage(i)
	}
	s.browserReadThrough = through
}

func (s *execution) restoreBrowserReads() {
	s.browserReadThrough = 0
	for i := len(s.messages) - 1; i >= 0; i-- {
		m := s.messages[i]
		if m.Role == "assistant" && !m.Incomplete {
			s.acknowledgeBrowserPages(i)
			return
		}
	}
}

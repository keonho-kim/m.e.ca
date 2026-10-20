// Purpose: Keep compaction requests and restored context limited to continuation work.
// Pattern: Projection regression tests.
// Usage: go test ./internal/agent/compact -run TestMemoryContextHasNo.
// Related: internal/agent/compact/context.go, internal/agent/compact/summarize.go.
package compact

import (
	"strings"
	"testing"
)

func TestMemoryContextHasNoStorageBoundary(t *testing.T) {
	m := Memory{Revision: 7, Through: 23456, Markdown: "- Implement the remaining route."}
	c := Controller{memory: m, Prompt: "Summarize work", Settings: Defaults(), ActiveRequest: "Finish the route", Redact: func(s string) string { return s }}
	got, _ := c.Context()
	if got != m.Markdown {
		t.Fatalf("memo contains storage metadata: %q", got)
	}
	sources := sourceBuffer{limit: 10000}
	if err := sources.add(Record{Sequence: 23456, Role: "user", Text: "Finish the route"}, c.Redact); err != nil {
		t.Fatal(err)
	}
	messages, err := c.summaryMessages(m.Markdown, sources.packet(), 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"EarlierMemoThrough", "SourceThrough", "RecentConversation", "Sequence", "23456"} {
		if strings.Contains(messages[1].Content, key) {
			t.Fatalf("compaction received %s", key)
		}
	}
}

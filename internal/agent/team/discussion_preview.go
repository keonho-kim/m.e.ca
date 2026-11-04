// Purpose: Keep a bounded latest public opinion per active teammate for post-compaction decisions.
// Pattern: Request-owned compact projection.
// Usage: Publication records a preview; coherent work snapshots render it without rereading SQLite.
// Related: internal/agent/team/publication.go, internal/agent/team/work_context.go.
package team

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/keonho-kim/meca/internal/agent/team/channel"
)

type discussionPreview struct {
	sequence, replyTo int64
	from, text        string
}

func previewDiscussion(value channel.Message) discussionPreview {
	text := value.Text
	if len(text) > 2048 {
		end := 2048
		for !utf8.RuneStart(text[end]) {
			end--
		}
		// A small current-state preview must not retain a large evicted original allocation.
		text = text[:end] + "… (full public note retained)"
	}
	return discussionPreview{sequence: value.Sequence, replyTo: value.Note.ReplyTo, from: value.From, text: text}
}

func writeDiscussions(out *strings.Builder, values []discussionPreview) {
	header := false
	for _, value := range values {
		if value.sequence == 0 {
			continue
		}
		if !header {
			out.WriteString("\nRecent peer feedback (opinions, not team agreement):\n")
			header = true
		}
		fmt.Fprintf(out, "#%d %s", value.sequence, value.from)
		if value.replyTo > 0 {
			fmt.Fprintf(out, " replying to #%d", value.replyTo)
		}
		out.WriteString(": ")
		out.WriteString(value.text)
		out.WriteByte('\n')
	}
}

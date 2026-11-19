// Purpose: Keep the visible reply stable across attempts at one model decision.
// Pattern: Decision-owned snapshot with an attempt-local prefix check.
// Usage: askResponse publishes compatible extensions; drive retains the draft only on terminal failure.
// Related: internal/agent/run/generate.go, internal/agent/run/loop.go, internal/commands/server/stream_projection.go.
package run

import "strings"

type replyOutput struct{ text, thinking string }

func (o replyOutput) bytes() int64 { return int64(len(o.text) + len(o.thinking)) }

func (o replyOutput) visible(hideThinking bool) replyOutput {
	if hideThinking {
		o.thinking = ""
	}
	return o
}

type replyDraft struct {
	presentation string
	output       replyOutput
}

type replyReplay struct{ catchingUp, different bool }

func (r *replyReplay) accepts(previous, next replyOutput) bool {
	if r.different {
		return false
	}
	if !r.catchingUp {
		return true
	}
	if len(next.text) < len(previous.text) || len(next.thinking) < len(previous.thinking) {
		return false
	}
	// Compare the retained prefixes once when both channels catch up.
	r.catchingUp = false
	r.different = !strings.HasPrefix(next.text, previous.text) || !strings.HasPrefix(next.thinking, previous.thinking)
	return !r.different
}

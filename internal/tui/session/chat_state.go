// Purpose: Own conversation presentation, visible history and runtime text independently of session I/O.
// Pattern: Event-loop-owned feature state.
// Usage: Session routes worker notices; chatState applies transcript changes and token measurements.
// Related: internal/tui/session/runtime_events.go, internal/tui/session/chat_projection.go, internal/tui/session/tokens.go.
package session

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	"github.com/keonho-kim/meca/internal/tui/transcript"
	"github.com/keonho-kim/meca/internal/tui/viewport"
)

type chatState struct {
	retry                           retryNotice
	details                         detailRead
	toolActivity                    toolActivity
	chatVisible, selectedTranscript []viewport.Line
	historyLoading, historyEnd      bool
	chatProjection                  chatProjection
	tokensPerSecond                 *float64
	measured, currentMeasurement    runrecord.Throughput
	tokenTotal, tokenCurrent        modelapi.TokenUsage
	runSummary                      *viewport.Block
	output                          viewport.View
	stream, thinking                string
	historyCursor                   int64
	historyRevision                 int
}

func (c *chatState) receiveTranscript(e *runrecord.Event, formatter transcript.Formatter) {
	if e.Statistics != nil {
		blocks := formatter.Event(*e)
		c.runSummary = &blocks[0]
		return
	}
	switch e.Kind {
	case "progress":
		c.stream, c.thinking = "", ""
		c.output.AppendBlock(transcript.Answer(e.Text))
	case "thinking":
		c.thinking = ""
		c.output.AppendBlock(transcript.Thought(e.Thinking))
	case "stream":
		c.stream, c.thinking = e.Text, e.Thinking
	case "response":
		c.stream, c.thinking = "", ""
		c.output.AppendBlock(transcript.Thought(e.Thinking))
		c.output.AppendBlock(transcript.Answer(e.Text))
	default:
		c.output.Append(e.Text, false)
	}
}

//go:build !tui

// Purpose: Consume benchmark SSE output through the real incremental terminal renderer without accumulation.
// Pattern: Bounded synchronous frontend fixture for the complete engine workload.
// Usage: The workload handler writes framed events here; terminal events confirm successful completion.
// Related: cmd/meca/engine_workload_test.go, internal/tui/markdown/stream.go.
package main

import (
	"bytes"
	"fmt"
	"strings"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/tui/markdown"
	"google.golang.org/protobuf/encoding/protojson"
)

type engineStream struct {
	buffer    []byte
	text      strings.Builder
	render    markdown.Stream
	err       error
	completed bool
}

func (s *engineStream) Write(body []byte) (int, error) {
	if s.err != nil {
		return len(body), nil
	}
	if len(s.buffer)+len(body) > 1<<20 {
		s.err = fmt.Errorf("fixture stream frame exceeded its bound")
		return len(body), nil
	}
	s.buffer = append(s.buffer, body...)
	offset := 0
	for {
		end := bytes.Index(s.buffer[offset:], []byte("\n\n"))
		if end < 0 {
			break
		}
		s.consume(s.buffer[offset : offset+end])
		offset += end + 2
	}
	copy(s.buffer, s.buffer[offset:])
	s.buffer = s.buffer[:len(s.buffer)-offset]
	return len(body), nil
}

func (s *engineStream) consume(frame []byte) {
	for line := range bytes.SplitSeq(frame, []byte("\n")) {
		data, ok := bytes.CutPrefix(line, []byte("data: "))
		if !ok {
			continue
		}
		var notification v1.Notification
		if err := protojson.Unmarshal(data, &notification); err != nil {
			s.err = err
			return
		}
		event := notification.GetEvent()
		if event == nil {
			continue
		}
		switch event.Kind {
		case "draft_reset":
			s.text.Reset()
			s.render = markdown.Stream{}
		case "assistant_delta":
			s.text.WriteString(event.Text)
			rows := s.render.Render(s.text.String(), 100)
			for _, row := range rows[max(0, len(rows)-24):] {
				for range row.RowsFrom(100, 0, "") {
				}
			}
		case "request_finished":
			s.completed = true
			if event.Status != "answered" {
				s.err = fmt.Errorf("native fixture task failed: %s %s", event.Status, event.Error)
			}
		}
	}
}

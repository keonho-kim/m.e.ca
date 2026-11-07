// Purpose: Consume benchmark SSE output through the real incremental terminal renderer without accumulation.
// Pattern: Bounded synchronous frontend fixture for the complete engine workload.
// Usage: The workload handler writes framed events here; terminal events confirm successful completion.
// Related: cmd/meca/engine_workload_test.go, internal/tui/markdown/stream.go.
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/keonho-kim/meca/internal/tui/markdown"
)

type engineStream struct {
	header    http.Header
	status    int
	buffer    []byte
	text      strings.Builder
	render    markdown.Stream
	err       error
	failure   string
	completed bool
}

func (s *engineStream) Header() http.Header    { return s.header }
func (s *engineStream) WriteHeader(status int) { s.status = status }
func (s *engineStream) Flush()                 {}
func (s *engineStream) Write(body []byte) (int, error) {
	if s.err != nil {
		return len(body), nil
	}
	if len(s.buffer)+len(body) > 1<<20 {
		s.err = fmt.Errorf("fixture stream frame exceeded its bound")
		return len(body), nil
	}
	if s.status >= 400 {
		s.failure = string(body[:min(len(body), 4096)])
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
		var event struct{ Type, Delta string }
		if err := json.Unmarshal(data, &event); err != nil {
			s.err = err
			return
		}
		switch event.Type {
		case "response.output_text.delta":
			s.text.WriteString(event.Delta)
			rows := s.render.Render(s.text.String(), 100)
			for _, row := range rows[max(0, len(rows)-24):] {
				for range row.RowsFrom(100, 0, "") {
				}
			}
		case "response.completed":
			s.completed = true
		case "response.failed", "error":
			s.err = fmt.Errorf("fixture stream failed: %s", data[:min(len(data), 4096)])
		}
	}
}

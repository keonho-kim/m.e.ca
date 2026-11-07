// Purpose: Turn native draft generations into append-only public text with verified final equality.
// Pattern: Streaming projection with explicit rewrite failure.
// Usage: Compatible SSE observers own this state; it never mutates engine execution.
// Related: internal/serverapi/openai/stream.go, internal/commands/server/projection.go.
package openai

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"strings"
	"unicode"

	"github.com/keonho-kim/meca/internal/commands/server"
)

type outputProjection struct {
	all, segment        hash.Hash
	bytes, segmentBytes int
	pending             strings.Builder
	committed, buffered bool
	emit                func(string) error
}

func newOutput(buffered bool, emit func(string) error) *outputProjection {
	return &outputProjection{all: sha256.New(), segment: sha256.New(), buffered: buffered, emit: emit}
}
func (p *outputProjection) append(text string) error {
	if text == "" {
		return nil
	}
	if p.segmentBytes == 0 && p.bytes > 0 {
		if err := p.emit("\n\n"); err != nil {
			return err
		}
		_, _ = p.all.Write([]byte("\n\n"))
		p.bytes += 2
	}
	if err := p.emit(text); err != nil {
		return err
	}
	body := []byte(text)
	_, _ = p.all.Write(body)
	_, _ = p.segment.Write(body)
	p.bytes += len(text)
	p.segmentBytes += len(text)
	return nil
}
func (p *outputProjection) event(event server.Event) error {
	if !p.accepts(event) {
		return nil
	}
	switch event.Record.Kind {
	case "draft_reset":
		if p.segmentBytes > 0 && !p.committed {
			return fmt.Errorf("generation rewrote an exposed draft")
		}
		p.segment.Reset()
		p.segmentBytes = 0
		p.pending.Reset()
		p.committed = false
	case "assistant_delta":
		text := event.Record.Text
		if p.segmentBytes == 0 {
			text = strings.TrimLeftFunc(text, unicode.IsSpace)
		}
		body := strings.TrimRightFunc(text, unicode.IsSpace)
		if body != "" {
			prefix := p.pending.String()
			p.pending.Reset()
			if err := p.append(prefix + body); err != nil {
				return err
			}
		}
		trailing := text[len(body):]
		if p.pending.Len()+len(trailing) > 64<<10 {
			p.buffered = true
			p.pending.Reset()
			return nil
		}
		p.pending.WriteString(trailing)
	case "progress", "response":
		if event.Truncated {
			// A clipped commit cannot prove its missing suffix. Continue from
			// the original journal at completion, without inventing any bytes.
			p.buffered = true
			p.pending.Reset()
			return nil
		}
		text := strings.TrimSpace(event.Record.Text)
		if !matchesPrefix(p.segment, p.segmentBytes, text) {
			return fmt.Errorf("accepted generation differs from its exposed draft")
		}
		p.pending.Reset()
		if err := p.append(text[p.segmentBytes:]); err != nil {
			return err
		}
		p.committed = true
	}
	return nil
}

func (p *outputProjection) accepts(event server.Event) bool {
	if p.buffered || event.Record.Agent != "" {
		return false
	}
	switch event.Record.Kind {
	case "draft_reset", "assistant_delta", "progress", "response":
		return true
	}
	return false
}
func matchesPrefix(h hash.Hash, n int, text string) bool {
	if n > len(text) {
		return false
	}
	sum := sha256.Sum256([]byte(text[:n]))
	return bytes.Equal(sum[:], h.Sum(nil))
}
func (p *outputProjection) finish(text string) error {
	if !matchesPrefix(p.all, p.bytes, text) {
		return fmt.Errorf("final output differs from its exposed draft")
	}
	if p.bytes < len(text) {
		return p.emit(text[p.bytes:])
	}
	return nil
}

// Purpose: Preserve stream content across fragmented, multiline and oversized frames.
// Pattern: Transport boundary and retained-buffer regression tests.
// Usage: Protocol tests use in-memory readers without network or model calls.
// Related: internal/provider/protocol/lines.go, internal/provider/protocol/stream.go.
package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type fragmentedReader struct {
	io.Reader
	chunk int
}

func (r fragmentedReader) Read(p []byte) (int, error) { return r.Reader.Read(p[:min(len(p), r.chunk)]) }

func TestStreamFramesAcrossFragmentationAndLongLines(t *testing.T) {
	text := strings.Repeat("value 한글 ", 10000)
	raw, _ := json.Marshal(text)
	for _, ending := range []string{"\n", "\r\n"} {
		wire := "data: {\"choices\":[{\"delta\":{\"content\":" + string(raw) + "}}]}" + ending + ending +
			": heartbeat" + ending + "data: {\"choices\":[" + ending + "data: {\"delta\":{\"content\":\"tail\"}}]}" + ending + ending + "data: [DONE]"
		for _, chunk := range []int{1, 7, 4096, len(wire)} {
			response := &http.Response{Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(fragmentedReader{strings.NewReader(wire), chunk})}
			var got strings.Builder
			err := ReadResponse(context.Background(), response, connection.VLLM, func(e modelapi.Event) error { got.WriteString(e.Delta); return nil })
			if err != nil || got.String() != text+"tail" {
				t.Fatalf("chunk=%d: content changed: %v", chunk, err)
			}
		}
	}
}

func TestStreamLinesReleaseExceptionalBuffer(t *testing.T) {
	r := streamLines{Reader: bufio.NewReader(strings.NewReader(strings.Repeat("x", 2*retainedFrameBytes) + "\nshort\n"))}
	if line, err := r.next(context.Background()); err != nil || len(line) != 2*retainedFrameBytes+1 {
		t.Fatal("long accepted line lost", err)
	}
	if line, err := r.next(context.Background()); err != nil || string(line) != "short\n" || cap(r.long) > retainedFrameBytes {
		t.Fatal("exceptional storage retained after the next line", err)
	}
}

// Purpose: Read bounded native NDJSON, compatible SSE and single JSON responses.
// Pattern: Adapter stream boundary.
// Usage: Called by Client.Generate; closes with its caller's HTTP response lifecycle.
// Related: internal/provider/httpclient/chunk.go, internal/provider/httpclient/client_test.go, internal/agent/model.go.

package httpclient

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/keonho-kim/v-code/internal/agent"
)

func readResponse(ctx context.Context, response *http.Response, native bool, emit func(agent.ModelEvent) error) error {
	state := streamState{}
	reader := bufio.NewReader(io.LimitReader(response.Body, 8*1024*1024))
	sse := strings.Contains(response.Header.Get("Content-Type"), "text/event-stream")
	state.streaming = sse || native && strings.Contains(response.Header.Get("Content-Type"), "ndjson")
	if !native && !sse {
		content, err := io.ReadAll(io.LimitReader(reader, agent.MaxResponseBytes+1))
		if err != nil {
			return err
		}
		if len(content) > agent.MaxResponseBytes {
			return fmt.Errorf("response exceeds size limit")
		}
		if err := state.consume(content, false, true, emit); err != nil {
			return err
		}
		return state.complete(emit)
	}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	var frame bytes.Buffer
	consumeFrame := func() error {
		if frame.Len() == 0 {
			return nil
		}
		content := bytes.TrimSpace(frame.Bytes())
		if bytes.Equal(content, []byte("[DONE]")) {
			state.finished = true
			frame.Reset()
			return nil
		}
		err := state.consume(content, false, false, emit)
		frame.Reset()
		return err
	}
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		line := scanner.Text()
		if native {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if err := state.consume([]byte(line), true, false, emit); err != nil {
				return err
			}
		} else {
			if line == "" {
				if err := consumeFrame(); err != nil {
					return err
				}
			} else if strings.HasPrefix(line, "data:") {
				frame.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
				frame.WriteByte('\n')
				if frame.Len() > 1024*1024 {
					return fmt.Errorf("SSE frame exceeds size limit")
				}
			}
		}
		if state.finished {
			return state.complete(emit)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read model stream: %w", err)
	}
	if !native && frame.Len() > 0 {
		if err := consumeFrame(); err != nil {
			return err
		}
	}
	return state.complete(emit)
}

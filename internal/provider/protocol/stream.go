// Purpose: Read streaming NDJSON and SSE without transport byte caps.
// Pattern: Adapter stream boundary.
// Usage: Called by Client.Generate; closes with its caller's HTTP response lifecycle.
// Related: internal/provider/protocol/chunk.go, internal/provider/httpclient/client_test.go, internal/agent/model/model.go, internal/provider/connection/connection.go.

package protocol

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
	"io"
	"net/http"
	"strings"
)

func ReadResponse(ctx context.Context, response *http.Response, provider modelconn.ProviderKind, emit func(modelapi.Event) error) error {
	state := streamState{provider: provider}
	native := provider == modelconn.Ollama
	reader := bufio.NewReader(response.Body)
	sse := strings.Contains(response.Header.Get("Content-Type"), "text/event-stream")
	state.streaming = sse || native && strings.Contains(response.Header.Get("Content-Type"), "ndjson")
	if !native && !sse {
		content, err := io.ReadAll(io.LimitReader(reader, modelapi.MaxResponseBytes+1))
		if err != nil {
			return err
		}
		if len(content) > modelapi.MaxResponseBytes {
			return fmt.Errorf("response exceeds size limit")
		}
		if err := state.consume(content, true, emit); err != nil {
			return err
		}
		return state.complete(emit)
	}
	var frame bytes.Buffer
	consumeFrame := func() error {
		if frame.Len() == 0 {
			return nil
		}
		content := bytes.TrimSpace(frame.Bytes())
		if bytes.Equal(content, []byte("[DONE]")) {
			if provider == modelconn.LMStudio && !state.finished {
				return fmt.Errorf("Responses stream ended without a completed response")
			}
			state.finished = true
			frame.Reset()
			return nil
		}
		err := state.consume(content, false, emit)
		frame.Reset()
		return err
	}
	for {
		line, readErr := reader.ReadString('\n')
		if readErr != nil && readErr != io.EOF {
			return fmt.Errorf("read model stream: %w", readErr)
		}
		if readErr == io.EOF && len(line) == 0 {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if native {
			if strings.TrimSpace(line) == "" {
				continue
			}
			if err := state.consume([]byte(line), false, emit); err != nil {
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
			}
		}
		if state.finished {
			return state.complete(emit)
		}
		if readErr == io.EOF {
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !native && frame.Len() > 0 {
		if err := consumeFrame(); err != nil {
			return err
		}
	}
	return state.complete(emit)
}

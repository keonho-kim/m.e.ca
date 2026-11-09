// Purpose: Bound and decode ordered NDJSON and SSE before forwarding model events.
// Pattern: Adapter stream boundary.
// Usage: Called by Client.Generate; closes with its caller's HTTP response lifecycle.
// Related: internal/provider/protocol/chunk.go, internal/provider/httpclient/client_test.go, internal/agent/model/model.go, internal/provider/connection/connection.go.

package protocol

import (
	"bufio"
	"bytes"
	"context"

	"io"
	"net/http"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	modelconn "github.com/keonho-kim/meca/internal/provider/connection"
)

func ReadResponse(ctx context.Context, response *http.Response, provider modelconn.ProviderKind, emit func(modelapi.Event) error) error {
	return readResponse(ctx, response, provider, modelapi.MaxResponseBytes, emit)
}

func readResponse(ctx context.Context, response *http.Response, provider modelconn.ProviderKind, frameLimit int, emit func(modelapi.Event) error) error {
	if err := context.Cause(ctx); err != nil {
		return err
	}
	state := streamState{provider: provider}
	native := provider == modelconn.Ollama
	reader := streamLines{Reader: bufio.NewReader(response.Body), limit: frameLimit}
	sse := strings.Contains(response.Header.Get("Content-Type"), "text/event-stream")
	state.Streaming = sse || native && strings.Contains(response.Header.Get("Content-Type"), "ndjson")
	if !native && !sse {
		content, err := io.ReadAll(io.LimitReader(reader.Reader, int64(frameLimit)+1))
		if err != nil {
			return err
		}
		if len(content) > frameLimit {
			return faults.Errorf(diagcatalog.ProtocolResponseExceedsSizeLimit)
		}
		if err := state.consume(content, true, emit); err != nil {
			return err
		}
		return state.complete(emit)
	}
	var frame bytes.Buffer
	consumeFrame := func(content []byte) error {
		if len(content) > frameLimit {
			return &modelapi.OutputLimitError{Reason: "model stream frame byte limit"}
		}
		if len(content) == 0 {
			return nil
		}
		content = bytes.TrimSpace(content)
		if bytes.Equal(content, []byte("[DONE]")) {
			if (modelconn.UsesResponses(provider) || provider == modelconn.Claude || provider == modelconn.Gemini) && !state.Finished {
				return faults.Errorf(diagcatalog.ProtocolModelResponsesStreamEndedWithoutACompleted)
			}
			state.Finished = true
			return nil
		}
		return state.consume(content, false, emit)
	}
	flushFrame := func() error {
		err := consumeFrame(frame.Bytes())
		if frame.Cap() > retainedFrameBytes {
			frame = bytes.Buffer{}
		} else {
			frame.Reset()
		}
		return err
	}
	for {
		line, readErr := reader.next(ctx)
		if readErr != nil && readErr != io.EOF {
			return faults.Errorf(diagcatalog.ProtocolReadModelStream, readErr)
		}
		if readErr == io.EOF && len(line) == 0 {
			break
		}
		if err := context.Cause(ctx); err != nil {
			return err
		}
		line = bytes.TrimSuffix(bytes.TrimSuffix(line, []byte("\n")), []byte("\r"))
		if native {
			if len(bytes.TrimSpace(line)) == 0 {
				continue
			}
			if err := state.consume(line, false, emit); err != nil {
				return err
			}
		} else {
			if len(line) == 0 {
				if err := flushFrame(); err != nil {
					return err
				}
			} else if bytes.HasPrefix(line, []byte("data:")) {
				data := bytes.TrimSpace(line[len("data:"):])
				if frame.Len() == 0 && len(data) > 0 && reader.skipBoundary() {
					if err := consumeFrame(data); err != nil {
						return err
					}
				} else {
					if len(data)+frame.Len()+1 > frameLimit {
						return &modelapi.OutputLimitError{Reason: "model stream frame byte limit"}
					}
					frame.Write(data)
					frame.WriteByte('\n')
				}
			}
		}
		if state.Finished {
			return state.complete(emit)
		}
		if readErr == io.EOF {
			break
		}
	}
	if err := context.Cause(ctx); err != nil {
		return err
	}
	if !native && frame.Len() > 0 {
		if err := flushFrame(); err != nil {
			return err
		}
	}
	return state.complete(emit)
}

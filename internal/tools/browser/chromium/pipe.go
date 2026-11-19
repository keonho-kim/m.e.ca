// Purpose: Exchange bounded DevTools messages with Chromium over inherited pipes.
// Pattern: Single caller with a joined stream reader and explicit command correlation.
// Usage: The read-only renderer uses a private pipe and never exposes a debugging port.
// Related: internal/tools/browser/chromium/process_unix.go, internal/tools/browser/chromium/render.go.
package chromium

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const maxFrame = 8 << 20

type message struct {
	ID        int
	Method    string
	SessionID string
	Result    json.RawMessage
	Params    json.RawMessage
	Error     json.RawMessage
}

type incoming struct {
	message message
	err     error
}

type pipe struct {
	ctx        context.Context
	input      *os.File
	output     *os.File
	messages   chan incoming
	readerDone chan struct{}
	nextID     int
}

func newPipe(ctx context.Context, input, output *os.File) *pipe {
	p := &pipe{ctx: ctx, input: input, output: output, messages: make(chan incoming), readerDone: make(chan struct{})}
	go p.read()
	return p
}

func (p *pipe) read() {
	defer close(p.readerDone)
	defer close(p.messages)
	r := bufio.NewReaderSize(p.output, 32<<10)
	for {
		wire, err := readFrame(r)
		var frame message
		if err == nil {
			err = json.Unmarshal(wire, &frame)
		}
		select {
		case p.messages <- incoming{message: frame, err: err}:
		case <-p.ctx.Done():
			return
		}
		if err != nil {
			return
		}
	}
}

func readFrame(r *bufio.Reader) ([]byte, error) {
	data, err := r.ReadSlice(0)
	if err == nil {
		return data[:len(data)-1], nil
	}
	if !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}
	frame := append([]byte(nil), data...)
	for {
		data, err = r.ReadSlice(0)
		if len(frame)+len(data) > maxFrame {
			return nil, fault.Errorf(diag.WebFetchBodyLimit)
		}
		frame = append(frame, data...)
		if err == nil {
			return frame[:len(frame)-1], nil
		}
		if !errors.Is(err, bufio.ErrBufferFull) {
			return nil, err
		}
	}
}

func (p *pipe) send(session, method string, params any) (int, error) {
	if err := p.ctx.Err(); err != nil {
		return 0, err
	}
	p.nextID++
	data, err := json.Marshal(struct {
		ID        int    `json:"id"`
		Method    string `json:"method"`
		SessionID string `json:"sessionId,omitempty"`
		Params    any    `json:"params,omitempty"`
	}{p.nextID, method, session, params})
	if err != nil {
		return 0, err
	}
	_, err = p.input.Write(append(data, 0))
	if err != nil {
		if p.ctx.Err() != nil {
			return 0, p.ctx.Err()
		}
		return 0, fault.Errorf(diag.WebFetchBrowserPipe, err)
	}
	return p.nextID, nil
}

func (p *pipe) result(item incoming, ok bool) (message, error) {
	if err := p.ctx.Err(); err != nil {
		return message{}, err
	}
	if !ok {
		return message{}, fault.Errorf(diag.WebFetchBrowserPipe, io.EOF)
	}
	if item.err != nil {
		return message{}, fault.Errorf(diag.WebFetchBrowserPipe, item.err)
	}
	return item.message, nil
}

func (p *pipe) call(session, method string, params, result any, event func(message) error) error {
	id, err := p.send(session, method, params)
	if err != nil {
		return err
	}
	for {
		select {
		case <-p.ctx.Done():
			return p.ctx.Err()
		case item, ok := <-p.messages:
			frame, err := p.result(item, ok)
			if err != nil {
				return err
			}
			if frame.ID == id {
				if len(frame.Error) != 0 {
					return fault.Errorf(diag.WebFetchBrowserCDP, method)
				}
				if result != nil {
					return json.Unmarshal(frame.Result, result)
				}
				return nil
			}
			if frame.Method != "" && event != nil {
				if err := event(frame); err != nil {
					return err
				}
			}
		}
	}
}

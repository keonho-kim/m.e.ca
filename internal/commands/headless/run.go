// Purpose: Execute one normal agent request with text or JSONL presentation and no terminal.
// Pattern: Command use case with an output adapter.
// Usage: cmd/v supplies the same request dependencies used by the TUI.
// Related: internal/agent/session/request.go, cmd/v/headless.go.
package headless

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/keonho-kim/v-code/internal/agent/run"
	sessionrun "github.com/keonho-kim/v-code/internal/agent/session"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"io"
	"strings"
)

func Run(ctx context.Context, request sessionrun.Request, output io.Writer, jsonMode bool) error {
	if request.Runner.Redact == nil {
		return fmt.Errorf("a secret redactor is required")
	}
	encoder := json.NewEncoder(output)
	if jsonMode {
		info := struct {
			Kind      string `json:"kind"`
			SessionID string `json:"session_id"`
			Project   string `json:"project_root"`
		}{"session", request.Runner.Session, request.Runner.Redact(request.Runner.ProjectRoot)}
		if err := encoder.Encode(info); err != nil {
			return err
		}
	}
	var text, thinking string
	emit := func(event run.Event) error {
		if !jsonMode {
			return nil
		}
		if event.Kind == "generation_started" {
			text, thinking = "", ""
		}
		if event.Kind == "stream" {
			nextText, nextThinking := event.Text, event.Thinking
			if strings.HasPrefix(nextText, text) {
				event.Text = strings.TrimPrefix(nextText, text)
			}
			if strings.HasPrefix(nextThinking, thinking) {
				event.Thinking = strings.TrimPrefix(nextThinking, thinking)
			}
			text, thinking = nextText, nextThinking
			event.Kind = "assistant_delta"
		}
		return encoder.Encode(event)
	}
	approve := func(context.Context, toolapi.Approval) (bool, error) {
		return false, fmt.Errorf("approval required in noninteractive mode; no operation was approved")
	}
	result, err := sessionrun.Execute(ctx, request, approve, emit)
	if jsonMode {
		finish := struct {
			Kind   string     `json:"kind"`
			Status run.Status `json:"status"`
			Error  string     `json:"error,omitempty"`
		}{Kind: "run_finished", Status: result.Status}
		if err != nil {
			finish.Error = request.Runner.Redact(err.Error())
		}
		err = errors.Join(err, encoder.Encode(finish))
	} else if result.Text != "" {
		_, writeErr := fmt.Fprintln(output, result.Text)
		err = errors.Join(err, writeErr)
	}
	if err == nil && result.Status != run.Answered {
		err = fmt.Errorf("request ended with status %s", result.Status)
	}
	return err
}

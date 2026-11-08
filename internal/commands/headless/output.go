// Purpose: Serialize ordered source-attributed JSONL and the final text for either execution mode.
// Pattern: Presentation adapter.
// Usage: Single Run and cooperative RunTeam own one output for their request.
// Related: internal/commands/headless/run.go, internal/commands/headless/team.go.
package headless

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type delta struct{ text, thinking string }
type output struct {
	writer   io.Writer
	encoder  *json.Encoder
	jsonMode bool
	single   delta
	peers    map[string]delta
	redact   func(string) string
}

func newOutput(writer io.Writer, jsonMode bool, redact func(string) string) *output {
	return &output{writer: writer, encoder: json.NewEncoder(writer), jsonMode: jsonMode, redact: redact}
}
func (o *output) begin(session, project string) error {
	if !o.jsonMode {
		return nil
	}
	return o.encoder.Encode(struct {
		Kind    string `json:"kind"`
		Session string `json:"session_id"`
		Project string `json:"project_root"`
	}{"session", session, o.redact(project)})
}
func (o *output) emit(event runrecord.Event) error {
	if !o.jsonMode {
		return nil
	}
	event.Kind = event.PublicKind()
	if event.Skill != nil {
		event.Output = nil
	}
	state := o.single
	if event.Agent != "" {
		state = o.peers[event.Agent]
	}
	if event.Kind == "generation_started" {
		state = delta{}
	}
	if event.Kind == "stream" {
		next := delta{event.Text, event.Thinking}
		if strings.HasPrefix(next.text, state.text) {
			event.Text = strings.TrimPrefix(next.text, state.text)
		}
		if strings.HasPrefix(next.thinking, state.thinking) {
			event.Thinking = strings.TrimPrefix(next.thinking, state.thinking)
		}
		state = next
		event.Kind = "assistant_delta"
	}
	if event.Agent != "" {
		if o.peers == nil {
			o.peers = map[string]delta{}
		}
		if event.Kind == "team_notice" && event.Text == "left" {
			delete(o.peers, event.Agent)
		} else {
			o.peers[event.Agent] = state
		}
	} else {
		o.single = state
	}
	return o.encoder.Encode(event)
}
func (o *output) finish(status runrecord.Status, text string, cause error) error {
	if o.jsonMode {
		event := struct {
			Kind   string           `json:"kind"`
			Status runrecord.Status `json:"status"`
			Error  string           `json:"error,omitempty"`
		}{Kind: "run_finished", Status: status}
		if cause != nil {
			event.Error = o.redact(cause.Error())
		}
		cause = errors.Join(cause, o.encoder.Encode(event))
	} else if text != "" {
		_, err := fmt.Fprintln(o.writer, text)
		cause = errors.Join(cause, err)
	}
	if cause == nil && status != runrecord.Answered {
		cause = faults.Errorf(diagcatalog.HeadlessRequestEndedWithStatus, status)
	}
	return cause
}

// Purpose: Script isolated LAT phases and track their read-only tool lifetimes.
// Pattern: Per-run generator and executor fixtures.
// Usage: LAT runtime tests select replies by their installed phase prompt.
// Related: internal/agent/run/lat_test.go, internal/agent/run/run_test.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/keonho-kim/meca/internal/agent/lat"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type latHandler func(context.Context, modelapi.Request, func(modelapi.Event) error) error

type latModel struct {
	mu       sync.Mutex
	replies  map[string][]string
	requests map[string][]modelapi.Request
	handlers map[string]latHandler
	scopes   map[string]int
	created  int
	released int
}

func (m *latModel) NewRunGenerator() (modelapi.Generator, func()) {
	m.mu.Lock()
	m.created++
	m.mu.Unlock()
	scope := &latScope{model: m}
	return scope, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		scope.closed = true
		m.released++
	}
}

func (*latModel) Generate(context.Context, modelapi.Request, func(modelapi.Event) error) error {
	return fmt.Errorf("LAT generation used the unscoped model")
}

type latScope struct {
	model  *latModel
	phase  string
	closed bool
}

func (s *latScope) Generate(ctx context.Context, request modelapi.Request, send func(modelapi.Event) error) error {
	m := s.model
	m.mu.Lock()
	if s.closed {
		m.mu.Unlock()
		return fmt.Errorf("generation after scope release")
	}
	phase := latPhase(request)
	if s.phase == "" {
		s.phase = phase
		m.scopes[phase]++
	} else if phase != s.phase {
		m.mu.Unlock()
		return fmt.Errorf("one generator scope served both %s and %s", s.phase, phase)
	}
	m.requests[phase] = append(m.requests[phase], request)
	index := len(m.requests[phase]) - 1
	handler := m.handlers[phase]
	replies := m.replies[phase]
	m.mu.Unlock()
	if handler != nil {
		return handler(ctx, request, send)
	}
	if index >= len(replies) {
		return fmt.Errorf("%s script exhausted at call %d", phase, index+1)
	}
	usage := &modelapi.TokenUsage{Input: 100, Output: 5, Cost: new(0.001)}
	value := replies[index]
	if strings.HasPrefix(value, "native:") {
		var call toolapi.Call
		if err := json.Unmarshal([]byte(strings.TrimPrefix(value, "native:")), &call); err != nil {
			return err
		}
		call.ID = fmt.Sprintf("%s_%d", phase, index+1)
		event := modelapi.Event{Calls: []toolapi.Call{call}, Done: true, Usage: usage}
		if phase == "main" {
			event.Thinking = "PRIVATE_PARENT_THINKING"
		}
		return send(event)
	}
	return send(modelapi.Event{Delta: value, Done: true, Usage: usage})
}

func latPhase(request modelapi.Request) string {
	for _, message := range request.Messages {
		for _, phase := range []string{"plan", "critique", "refine", "review"} {
			if strings.Contains(message.Content, "TEST_LAT_"+strings.ToUpper(phase)+"_PROMPT") {
				return phase
			}
		}
	}
	return "main"
}

func latFixture(replies ...string) (Runner, *latModel, *latTools, *journal) {
	main, tools, store := &generator{}, &latTools{executor: &executor{}}, &journal{}
	r := fixture(main, tools.executor, store)
	model := &latModel{
		replies: map[string][]string{
			"main": replies, "plan": {"INITIAL_PLAN"}, "critique": {"PLAN_CRITIQUE"},
			"refine": {"IMPROVED_PLAN"}, "review": {"NO_FINDINGS"},
		},
		requests: map[string][]modelapi.Request{}, handlers: map[string]latHandler{}, scopes: map[string]int{},
	}
	templates := prompt.Defaults()
	templates.LATPlan += "\nTEST_LAT_PLAN_PROMPT"
	templates.LATCritique += "\nTEST_LAT_CRITIQUE_PROMPT"
	templates.LATRefine += "\nTEST_LAT_REFINE_PROMPT"
	templates.LAT += "\nTEST_LAT_REVIEW_PROMPT"
	r.Generator, r.Tools, r.Journal, r.Prompts, r.LAT = model, tools, &latJournal{store}, &templates, true
	r.ReviewMemory = &journalMemory{journal: store}
	return r, model, tools, store
}

// Parent context projects durable LAT events alongside ordinary messages.
type latJournal struct{ *journal }

func (j *latJournal) ContextMessages(ctx context.Context, session string, after int64, maxBytes int, images bool) ([]modelapi.Message, error) {
	if after != 0 || len(j.timeline) == 0 {
		return j.journal.ContextMessages(ctx, session, after, maxBytes, images)
	}
	var messages []modelapi.Message
	size := 0
	for _, record := range j.timeline {
		var message modelapi.Message
		if record.Kind == "message" {
			message = record.Message
		} else if !record.Event.Internal && record.Event.Kind == "lat_plan" {
			message = lat.Plan(record.Event.Text)
		} else if !record.Event.Internal && record.Event.Kind == "lat_feedback" {
			message = lat.Feedback(record.Event.Text)
		} else {
			continue
		}
		size += len(message.Content) + len(message.Calls)
		if size > maxBytes {
			return nil, fmt.Errorf("conversation limit exceeded")
		}
		messages = append(messages, message)
	}
	return messages, nil
}

type latTools struct {
	*executor
	lifecycle []string
	views     []*latReadOnly
	failure   error
}

func (t *latTools) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	t.lifecycle = append(t.lifecycle, "main:"+call.Name)
	return t.executor.Prepare(ctx, settings, call)
}

func (t *latTools) ReadOnly() (toolapi.Executor, func() error, error) {
	if t.failure != nil {
		return nil, nil, t.failure
	}
	t.lifecycle = append(t.lifecycle, "open")
	view := &latReadOnly{parent: t, executor: &executor{}}
	t.views = append(t.views, view)
	return view, func() error {
		if view.closed {
			return fmt.Errorf("read-only view closed twice")
		}
		view.closed = true
		t.lifecycle = append(t.lifecycle, "close")
		return nil
	}, nil
}

type latReadOnly struct {
	parent   *latTools
	executor *executor
	closed   bool
}

func (t *latReadOnly) Definitions() []toolapi.Definition {
	var definitions []toolapi.Definition
	for _, definition := range t.executor.Definitions() {
		if definition.Effect == policy.Read {
			definitions = append(definitions, definition)
		}
	}
	return definitions
}

func (t *latReadOnly) Prepare(ctx context.Context, settings policy.Settings, call toolapi.Call) (toolapi.Execution, error) {
	if t.closed || call.Name != "inspect_fixture" {
		return nil, fmt.Errorf("read-only view rejected %s", call.Name)
	}
	t.parent.lifecycle = append(t.parent.lifecycle, "read:"+call.Name)
	return t.executor.Prepare(ctx, settings, call)
}

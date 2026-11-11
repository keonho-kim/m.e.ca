//go:build !tui

// Purpose: Reproduce intermediate responses around two real command executions for browser checks.
// Pattern: Request-cancelled, manually advanced local model fixture.
// Usage: uiProvider routes repeated-tool requests and private test controls here.
// Related: cmd/meca/ui_provider_test.go, webui/e2e/tool-presentation.spec.ts.
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type uiToolSequence struct {
	advance [3]chan struct{}
	once    [3]sync.Once
}

func newUIToolSequence() *uiToolSequence {
	s := &uiToolSequence{}
	for i := range s.advance {
		s.advance[i] = make(chan struct{})
	}
	return s
}

func (s *uiToolSequence) control(w http.ResponseWriter, r *http.Request) bool {
	const prefix = "/test/tool-sequence/"
	if !strings.HasPrefix(r.URL.Path, prefix) {
		return false
	}
	step, err := strconv.Atoi(strings.TrimPrefix(r.URL.Path, prefix))
	if err != nil || step < 1 || step > len(s.advance) {
		http.NotFound(w, r)
		return true
	}
	s.once[step-1].Do(func() { close(s.advance[step-1]) })
	w.WriteHeader(http.StatusNoContent)
	return true
}

func (s *uiToolSequence) reply(w http.ResponseWriter, r *http.Request, step int) {
	if step > 2 {
		http.Error(w, "unexpected fixture generation", http.StatusBadRequest)
		return
	}
	chunk := func(delta any, finish any) {
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"delta": delta, "finish_reason": finish}}})
		fmt.Fprintf(w, "data: %s\n\n", body)
		_ = http.NewResponseController(w).Flush()
	}
	thoughts := []string{"First command reasoning.", "Between commands.", "Final reasoning."}
	texts := []string{"Before first command.", "Before second command.", "Final response outside statistics."}
	chunk(map[string]string{"reasoning": thoughts[step]}, nil)
	chunk(map[string]string{"content": texts[step]}, nil)
	select {
	case <-s.advance[step]:
	case <-r.Context().Done():
		return
	}
	if step < 2 {
		arguments, _ := json.Marshal(map[string]string{"command": fmt.Sprintf("printf 'tool-%d'", step+1)})
		chunk(map[string]any{"tool_calls": []any{map[string]any{
			"index": 0, "id": "fixture-reused-command", "type": "function",
			"function": map[string]string{"name": "execute_command", "arguments": string(arguments)},
		}}}, "tool_calls")
	} else {
		chunk(map[string]string{}, "stop")
		fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":32,\"completion_tokens\":64}}\n\n")
	}
	fmt.Fprint(w, "data: [DONE]\n\n")
	_ = http.NewResponseController(w).Flush()
}

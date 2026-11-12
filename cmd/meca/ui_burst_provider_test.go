//go:build !tui

// Purpose: Provide a controlled short text burst for background-rendering browser contracts.
// Pattern: Request-cancellable model fixture with explicit start admission.
// Usage: uiProvider wires its control endpoint and response branch.
// Related: webui/e2e/presentation-budget.spec.ts, cmd/meca/ui_provider_test.go.
package main

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type uiTextBurst struct {
	start chan struct{}
	mu    sync.Mutex
}

func (b *uiTextBurst) control(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path != "/test/start-burst" {
		return false
	}
	b.mu.Lock()
	if b.start != nil {
		close(b.start)
		b.start = nil
	}
	b.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
	return true
}
func (b *uiTextBurst) reply(w http.ResponseWriter, r *http.Request, text string, chunk func(string, string, any)) bool {
	if !strings.Contains(text, "BURST_FIXTURE") {
		return false
	}
	start := make(chan struct{})
	b.mu.Lock()
	b.start = start
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		if b.start == start {
			b.start = nil
		}
		b.mu.Unlock()
	}()
	chunk("content", "Burst ready.\n\n", nil)
	select {
	case <-start:
	case <-r.Context().Done():
		return true
	}
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for index := range 60 {
		select {
		case <-ticker.C:
		case <-r.Context().Done():
			return true
		}
		// Changing usage beside text must not bypass the UI's presentation budget.
		fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Background presentation update. \"},\"finish_reason\":null}],\"usage\":{\"prompt_tokens\":12000,\"completion_tokens\":%d}}\n\n", index+1)
		_ = http.NewResponseController(w).Flush()
	}
	chunk("content", "\n\nCompleted burst.", "stop")
	fmt.Fprint(w, "data: {\"choices\":[],\"usage\":{\"prompt_tokens\":12000,\"completion_tokens\":6400}}\n\ndata: [DONE]\n\n")
	_ = http.NewResponseController(w).Flush()
	return true
}

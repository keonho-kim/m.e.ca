// Purpose: Keep independent latest-value stream mailboxes for concurrent peers and serialize approvals.
// Pattern: Source-keyed delivery adapter with ordered barriers.
// Usage: startRequest uses single delivery unchanged or the cooperative multiplexed path.
// Related: internal/tui/session/delivery.go, internal/tui/session/tasks.go.
package session

import (
	"context"
	"sync"

	runrecord "github.com/keonho-kim/meca/internal/agent/record"
)

type requestDelivery struct {
	team   bool
	single runtimeDelivery
	mu     sync.Mutex
	peers  map[string]*runtimeDelivery
	texts  map[string]string
}

func (d *requestDelivery) peer(id string) *runtimeDelivery {
	if d.peers == nil {
		d.peers = map[string]*runtimeDelivery{}
	}
	if d.peers[id] == nil {
		d.peers[id] = &runtimeDelivery{}
	}
	return d.peers[id]
}
func (d *requestDelivery) stream(ctx context.Context, queue chan<- notice, event runrecord.Event) error {
	if !d.team {
		return d.single.stream(ctx, queue, event)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	event.Thinking = ""
	if text, exists := d.texts[event.Agent]; exists && text == event.Text {
		return nil
	}
	if d.texts == nil {
		d.texts = map[string]string{}
	}
	d.texts[event.Agent] = event.Text
	return d.peer(event.Agent).stream(ctx, queue, event)
}
func (d *requestDelivery) post(ctx context.Context, queue chan<- notice, n notice) error {
	if n.Event != nil && (n.Event.Kind == "team_observation" || n.Event.Kind == "team_status") {
		return ctx.Err()
	}
	if !d.team {
		return d.single.post(ctx, queue, n)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	id := ""
	if n.Event != nil {
		id = n.Event.Agent
		if n.Event.Kind == "thinking" {
			return nil
		}
		if n.Event.Kind == "generation_started" {
			delete(d.texts, id)
		}
	}
	if n.Approval != nil || n.Done {
		for _, peer := range d.peers {
			peer.current, peer.usage = nil, nil
		}
	}
	err := d.peer(id).post(ctx, queue, n)
	if n.Event != nil && n.Event.Kind == "team_notice" && n.Event.Text == "left" {
		delete(d.peers, id)
		delete(d.texts, id)
	}
	return err
}

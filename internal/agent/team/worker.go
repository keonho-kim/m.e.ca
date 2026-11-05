// Purpose: Own each private participant lifetime and join its resources before releasing the room slot.
// Pattern: Joined worker lifecycle.
// Usage: Initial admission and invitations call start under memberMu.
// Related: internal/agent/team/room.go, internal/agent/team/answer.go.
package team

import (
	"errors"
	"fmt"
	"github.com/keonho-kim/meca/internal/agent/team/channel"
	"github.com/keonho-kim/meca/internal/agent/team/identity"
)

func (r *Room) start(member identity.Member) error {
	live := r.reserve(member)
	return r.launch(live)
}
func (r *Room) reserve(member identity.Member) *participant {
	live := &participant{member: member, wake: make(chan struct{}, 1)}
	progress := progressKey("member", member.ID, member.Session)
	r.mu.Lock()
	r.active[member.ID] = live
	r.invalidateWorkLocked()
	r.recordProgress(progress)
	r.mu.Unlock()
	return live
}
func (r *Room) launch(live *participant) error {
	member := live.member
	if err := r.notice(member.ID, "joined"); err != nil {
		r.mu.Lock()
		delete(r.active, member.ID)
		r.mu.Unlock()
		return err
	}
	r.workers.Add(1)
	go r.worker(live)
	return nil
}
func (r *Room) worker(live *participant) {
	defer r.workers.Done()
	peer := &Peer{ID: live.member.ID, room: r}
	defer func() {
		r.memberMu.Lock()
		r.mu.Lock()
		live.closed = true
		r.work.Release(peer.ID)
		r.invalidateWorkLocked()
		if r.active[peer.ID] == live {
			delete(r.active, peer.ID)
		}
		for _, other := range r.active {
			if !other.closed && !other.leaving {
				wake(other)
			}
		}
		r.mu.Unlock()
		r.memberMu.Unlock()
		if r.ctx.Err() == nil {
			_ = r.notice(peer.ID, "left")
		}
		r.notify()
	}()
	r.mu.Lock()
	input := r.input
	r.mu.Unlock()
	result, err := r.options.Execute(r.ctx, live.member, input, peer)
	if err == nil && result.Status != "" && result.Status != "answered" {
		err = fmt.Errorf("peer stopped: %s", result.End.Reason)
	}
	if err == nil && !result.Departed {
		r.mu.Lock()
		finished := r.finished
		r.mu.Unlock()
		if !finished {
			again, answerErr := peer.AfterAnswer(r.ctx, result.Text)
			err = answerErr
			if again && err == nil {
				err = fmt.Errorf("peer execution returned before admitting pending public conversation")
			}
		}
	}
	if err != nil {
		r.mu.Lock()
		r.failures = errors.Join(r.failures, fmt.Errorf("%s: %w", peer.ID, err))
		finalFailure := r.finished && r.result.Agent == peer.ID
		r.mu.Unlock()
		if finalFailure {
			r.fail(err)
			return
		}
		if r.ctx.Err() == nil {
			_, _ = r.publish(channel.Message{From: peer.ID, Text: "Session stopped: " + err.Error()})
		}
	}
}

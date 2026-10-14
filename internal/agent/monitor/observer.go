// Purpose: Observe runtime facts on a dedicated worker and persist them before publishing snapshots.
// Pattern: Observer with an owned, joined worker lifecycle.
// Usage: Goal execution owns New and Close; model code cannot author Monitor records.
// Related: internal/agent/run/monitor.go, internal/storage/sqlite/monitor.go.
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"
)

type Record struct {
	Sequence                                   int64
	GoalID, TaskID, Kind, CallID, Tool, Detail string
}
type Persist func(context.Context, Record) error
type command struct {
	ctx    context.Context
	record *Record
	reply  chan response
}
type response struct {
	text string
	err  error
}
type Observer struct {
	commands chan command
	done     chan struct{}
	cancel   context.CancelFunc
	once     sync.Once
}

func New(parent context.Context, seed []Record, persist Persist) *Observer {
	ctx, cancel := context.WithCancel(parent)
	observer := &Observer{commands: make(chan command), done: make(chan struct{}), cancel: cancel}
	seed = append([]Record(nil), seed...)
	go observer.run(ctx, seed, persist)
	return observer
}
func (r Record) Check() error {
	if r.Sequence < 0 || r.Kind == "" || len(r.Kind) > 64 || len(r.GoalID) > 128 || len(r.TaskID) > 128 || len(r.CallID) > 128 || len(r.Tool) > 128 || len(r.Detail) > 1200 || !utf8.ValidString(r.Detail) {
		return fmt.Errorf("invalid Monitor observation")
	}
	return nil
}
func Detail(text string) string {
	if len(text) <= 1024 {
		return text
	}
	end := 1024
	for !utf8.RuneStart(text[end]) {
		end--
	}
	return text[:end] + " [excerpt; see original runtime event]"
}
func (o *Observer) run(ctx context.Context, recent []Record, persist Persist) {
	defer close(o.done)
	var sequence int64
	for _, record := range recent {
		sequence = max(sequence, record.Sequence)
	}
	if len(recent) > 64 {
		recent = append([]Record(nil), recent[len(recent)-64:]...)
	}
	for {
		select {
		case <-ctx.Done():
			return
		case request := <-o.commands:
			if request.record == nil {
				body, err := json.Marshal(struct {
					Through int64
					Omitted int64
					Records []Record
				}{sequence, max(0, sequence-int64(len(recent))), recent})
				request.reply <- response{text: string(body), err: err}
				continue
			}
			record := *request.record
			record.Sequence = sequence + 1
			err := record.Check()
			if err == nil && persist == nil {
				err = fmt.Errorf("Monitor persistence is required")
			}
			if err == nil {
				writeCtx, cancel := context.WithCancel(request.ctx)
				stop := context.AfterFunc(ctx, cancel)
				err = persist(writeCtx, record)
				stop()
				cancel()
			}
			if err == nil {
				sequence = record.Sequence
				if len(recent) == 64 {
					copy(recent, recent[1:])
					recent[len(recent)-1] = Record{}
					recent = recent[:63]
				}
				recent = append(recent, record)
			}
			request.reply <- response{err: err}
		}
	}
}
func (o *Observer) send(ctx context.Context, record *Record) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	request := command{ctx: ctx, record: record, reply: make(chan response, 1)}
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-o.done:
		return "", fmt.Errorf("Monitor is closed")
	case o.commands <- request:
	}
	select {
	case result := <-request.reply:
		return result.text, result.err
	case <-ctx.Done():
		return "", ctx.Err()
	case <-o.done:
		return "", fmt.Errorf("Monitor is closed")
	}
}
func (o *Observer) Observe(ctx context.Context, record Record) error {
	record.Detail = Detail(strings.TrimSpace(record.Detail))
	_, err := o.send(ctx, &record)
	return err
}
func (o *Observer) Snapshot(ctx context.Context) (string, error) { return o.send(ctx, nil) }
func (o *Observer) Close()                                       { o.once.Do(o.cancel); <-o.done }

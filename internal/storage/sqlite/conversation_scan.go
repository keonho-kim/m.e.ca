// Purpose: Stream session metadata through bounded readers and one callback owner.
// Pattern: Request-scoped map/reduce with backpressure and joined cancellation.
// Usage: Conversations scans small catalogs serially and larger catalogs with two readers.
// Related: internal/storage/sqlite/conversation_reader.go, internal/storage/sqlite/conversations.go.
package sqlite

import (
	"context"
	"errors"
	"sync"
)

const conversationReaders = 2

type conversationFile struct{ path, id, group string }
type conversationScanResult struct {
	conversationResult
	err  error
	done bool
}

func (s *Sessions) scanConversations(ctx context.Context, project string, accept func(conversationResult)) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	var initial [16]conversationFile
	n := 0
	var scan *conversationScan
	err = s.walkDatabases(ctx, project, func(path, id, group string) error {
		job := conversationFile{path, id, group}
		if scan != nil {
			return scan.enqueue(job)
		}
		initial[n], n = job, n+1
		if n == len(initial) {
			scan = startConversationScan(ctx, initial[:], accept)
		}
		return nil
	})
	if scan != nil {
		return scan.finish(err)
	}
	if err != nil || n == 0 {
		return err
	}
	// Small catalogs allocate neither a queue nor workers. Their SQL retains
	// the caller's context, including its native fast path when it is uncancellable.
	reader, err := openConversationReader(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	for _, job := range initial[:n] {
		value, err := reader.Read(ctx, job)
		if err != nil {
			return err
		}
		if value.conversation.ID != "" {
			accept(value)
		}
	}
	return ctx.Err()
}

type conversationScan struct {
	ctx, parent context.Context
	cancel      context.CancelFunc
	jobs        chan conversationFile
	results     chan conversationScanResult
	workers     sync.WaitGroup
	finished    int
	failure     error
	accept      func(conversationResult)
}

func startConversationScan(parent context.Context, initial []conversationFile, accept func(conversationResult)) *conversationScan {
	ctx, cancel := context.WithCancel(parent)
	s := &conversationScan{ctx: ctx, parent: parent, cancel: cancel,
		jobs: make(chan conversationFile, 16), results: make(chan conversationScanResult, conversationReaders), accept: accept}
	for worker := range conversationReaders {
		s.workers.Go(func() {
			err := scanConversationFiles(ctx, worker, initial, s.jobs, s.results)
			s.results <- conversationScanResult{err: err, done: true}
		})
	}
	return s
}

func (s *conversationScan) receive(result conversationScanResult) {
	if result.done {
		s.finished++
	}
	if s.failure == nil && result.err != nil {
		s.failure = result.err
		s.cancel()
	}
	if s.failure == nil && !result.done && s.parent.Err() == nil {
		s.accept(result.conversationResult)
	}
}

func (s *conversationScan) enqueue(job conversationFile) error {
	for {
		if s.failure != nil {
			return s.failure
		}
		// Drain ready metadata while discovery is backpressured; callbacks and
		// the final summary array remain owned by the calling goroutine.
		select {
		case s.jobs <- job:
			return nil
		case result := <-s.results:
			s.receive(result)
		case <-s.ctx.Done():
			if s.failure != nil {
				return s.failure
			}
			return s.ctx.Err()
		}
	}
}

func (s *conversationScan) finish(cause error) error {
	if cause != nil {
		s.cancel()
	}
	close(s.jobs)
	for s.finished < conversationReaders {
		s.receive(<-s.results)
	}
	s.workers.Wait()
	s.cancel()
	if cause != nil {
		return errors.Join(s.parent.Err(), cause)
	}
	return errors.Join(s.parent.Err(), s.failure)
}

func scanConversationFiles(ctx context.Context, worker int, initial []conversationFile, jobs <-chan conversationFile, results chan<- conversationScanResult) (err error) {
	reader, err := openConversationReader(ctx)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, reader.Close()) }()
	read := func(job conversationFile) error {
		value, err := reader.Read(ctx, job)
		if err != nil {
			return err
		}
		if value.conversation.ID == "" {
			return nil
		}
		select {
		case results <- conversationScanResult{conversationResult: value}:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	for i := worker; i < len(initial); i += conversationReaders {
		if err := read(initial[i]); err != nil {
			return err
		}
	}
	for job := range jobs {
		if err := read(job); err != nil {
			return err
		}
	}
	return ctx.Err()
}

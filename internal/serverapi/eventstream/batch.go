// Purpose: Flush already available SSE batches without adding a timer, queue or retained body copy.
// Pattern: Single-writer scoped batching over HTTP's existing bounded output buffer.
// Usage: Native handlers batch received events and explicitly flush critical boundaries.
// Related: internal/serverapi/eventstream/writer.go, internal/serverapi/http.go.
package eventstream

// Batch flushes the first application frame immediately, then at 64KiB or the
// end of this synchronous producer. It never retries after a producer failure.
func (w *Writer) Batch(send func() error) error {
	if w.batch {
		return send()
	}
	w.batch = true
	err := send()
	w.batch = false
	if err != nil {
		return err
	}
	return w.Flush()
}

// Flush publishes pending application data. Control events and first output
// use it inside batches; no pending data means no extra network flush.
func (w *Writer) Flush() error {
	if !w.pending {
		return nil
	}
	if err := w.controller.Flush(); err != nil {
		return err
	}
	w.pending, w.sent, w.queued = false, true, 0
	w.idle.Activity()
	return nil
}

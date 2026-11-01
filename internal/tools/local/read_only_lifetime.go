// Purpose: Gate new read-only work and join active calls before releasing owned resources.
// Pattern: Short critical sections with a request-owned join barrier.
// Usage: The LAT executor enters each operation and closes only after all operations leave.
// Related: internal/tools/local/read_only.go, internal/tools/local/process/registry.go.
package local

import "fmt"

func (executor *readOnlyExecutor) enter() error {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.closed.Load() {
		return fmt.Errorf("LAT read-only executor is closed")
	}
	// Close uses this same gate before Wait, so Add cannot race a zero-count Wait.
	executor.running.Add(1)
	return nil
}

func (executor *readOnlyExecutor) close() error {
	executor.once.Do(func() {
		executor.mu.Lock()
		executor.closed.Store(true)
		executor.cancel()
		executor.mu.Unlock()
		executor.running.Wait()
		executor.mu.Lock()
		defer executor.mu.Unlock()
		for _, release := range executor.release {
			release()
		}
		executor.release = nil
		executor.closeErr = executor.Executor.Close()
	})
	return executor.closeErr
}

func (executor *readOnlyExecutor) Close() error { return executor.close() }

// AcknowledgeCommand releases completed command contexts after their durable receipt.
func (executor *readOnlyExecutor) AcknowledgeCommand(id string) error {
	executor.mu.Lock()
	defer executor.mu.Unlock()
	if executor.closed.Load() {
		return fmt.Errorf("LAT read-only executor is closed")
	}
	if err := executor.Executor.AcknowledgeCommand(id); err != nil {
		return err
	}
	if release := executor.release[id]; release != nil {
		release()
		delete(executor.release, id)
	}
	return nil
}

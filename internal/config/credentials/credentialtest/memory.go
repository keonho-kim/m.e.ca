// Purpose: Supply isolated cancellable key-store fixtures without using persistent key files.
// Pattern: In-memory credential I/O test double.
// Usage: Credential and TUI tests inject Memory into Vault.Keys.
// Related: internal/config/credentials/persistence.go, internal/tui/session/fixture_test.go.
package credentialtest

import (
	"context"
	"os"
	"sync"
)

type Memory struct {
	mu     sync.Mutex
	values map[string]string
}

func (m *Memory) Set(ctx context.Context, ref, value string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = map[string]string{}
	}
	m.values[ref] = value
	return nil
}
func (m *Memory) Get(ctx context.Context, ref string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.values[ref]
	if !ok {
		return "", os.ErrNotExist
	}
	return v, nil
}

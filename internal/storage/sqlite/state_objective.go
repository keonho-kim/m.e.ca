// Purpose: Reuse immutable checkpoint objectives without retaining their large input strings.
// Pattern: Store-owned single-fingerprint cache with transactional payload verification.
// Usage: Journal encoding shares the original objective payload across repeated run-state updates.
// Related: internal/storage/sqlite/run_state.go, internal/storage/sqlite/prepared_payload.go.
package sqlite

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"sync"

	json "github.com/goccy/go-json"
	"github.com/keonho-kim/meca/internal/storage/content"
)

type objectiveCache struct {
	mu          sync.Mutex
	run         string
	fingerprint [32]byte
	id          string
}

func (p *journalEncoder) objectivePayload(run, text string) (string, error) {
	fingerprint := objectiveFingerprint(text)
	cache := &p.store.objective
	cache.mu.Lock()
	id := ""
	if cache.run == run && cache.fingerprint == fingerprint {
		id = cache.id
	}
	cache.mu.Unlock()
	if id != "" {
		// Preparation may have been rolled back. A fingerprint never establishes
		// persistence: recheck the immutable payload through this transaction.
		physical, _ := content.Reference(id)
		digest, _ := hex.DecodeString(physical)
		value := &preparedPayload{storedValue: storedValue{id: physical}, digest: digest, known: true}
		err := p.lookup.QueryRowContext(p.ctx, "SELECT id FROM payloads WHERE digest=?", digest).Scan(&value.local)
		if err == nil {
			p.referenced = append(p.referenced, value)
			return id, nil
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return "", err
		}
	}
	raw, err := json.Marshal(text)
	if err != nil {
		return "", err
	}
	quote := json.Valid([]byte(text))
	if quote {
		raw = []byte(text)
	}
	if _, err := p.payload(raw); err != nil {
		return "", err
	}
	id = p.referenced[len(p.referenced)-1].id
	if quote {
		id = content.StringReference(id)
	}
	cache.mu.Lock()
	cache.run, cache.fingerprint, cache.id = run, fingerprint, id
	cache.mu.Unlock()
	return id, nil
}

func objectiveFingerprint(text string) [32]byte {
	digest := sha256.New()
	var buffer [32 << 10]byte
	for len(text) > 0 {
		n := copy(buffer[:], text)
		digest.Write(buffer[:n])
		text = text[n:]
	}
	var sum [32]byte
	copy(sum[:], digest.Sum(sum[:0]))
	return sum
}

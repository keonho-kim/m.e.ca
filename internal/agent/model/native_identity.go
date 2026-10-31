// Purpose: Fingerprint exclusively owned native history once without retaining duplicate byte buffers.
// Pattern: Immutable value with structural invalidation.
// Usage: Context projection seals completed messages; transport reuses their native fingerprints.
// Related: internal/agent/context/projection.go, internal/provider/protocol/source_history.go.
package model

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"
)

type nativeIdentity struct {
	calls, state      [32]byte
	arguments         []argumentIdentity
	continuation      *byte
	continuationBytes int
}

type argumentIdentity struct {
	id, name string
	data     *byte
	bytes    int
}

// SealHistory takes ownership of Calls and ReasoningState. Every consumer must
// keep their backing bytes read-only; changing bytes requires a new message.
// Value-level replacements invalidate fingerprints without copying old bodies.
func (m Message) SealHistory() Message {
	if len(m.Calls) == 0 && len(m.ReasoningState) == 0 {
		m.native = nil
		return m
	}
	calls, state := m.NativeFingerprint()
	identity := &nativeIdentity{calls: calls, state: state, continuation: byteAddress(m.ReasoningState), continuationBytes: len(m.ReasoningState)}
	if len(m.Calls) > 0 {
		identity.arguments = make([]argumentIdentity, len(m.Calls))
		for i, call := range m.Calls {
			identity.arguments[i] = argumentIdentity{call.ID, call.Name, byteAddress(call.Arguments), len(call.Arguments)}
		}
	}
	m.native = identity
	return m
}

// NativeFingerprint preserves defensive content hashing for unsealed inputs.
func (m Message) NativeFingerprint() (calls, state [32]byte) {
	owned := m.native
	if owned != nil && owned.matchesCalls(m) {
		calls = owned.calls
	} else if len(m.Calls) > 0 {
		h := sha256.New()
		for _, call := range m.Calls {
			hashPart(h, []byte(call.ID))
			hashPart(h, []byte(call.Name))
			hashPart(h, call.Arguments)
		}
		copy(calls[:], h.Sum(nil))
	}
	if owned != nil && owned.continuation == byteAddress(m.ReasoningState) && owned.continuationBytes == len(m.ReasoningState) {
		state = owned.state
	} else {
		state = sha256.Sum256(m.ReasoningState)
	}
	return calls, state
}

func (n *nativeIdentity) matchesCalls(m Message) bool {
	if len(n.arguments) != len(m.Calls) {
		return false
	}
	for i, call := range m.Calls {
		p := n.arguments[i]
		if p.id != call.ID || p.name != call.Name || p.bytes != len(call.Arguments) || p.data != byteAddress(call.Arguments) {
			return false
		}
	}
	return true
}

func byteAddress(data []byte) *byte {
	if len(data) == 0 {
		return nil
	}
	return &data[0]
}

func hashPart(h hash.Hash, raw []byte) {
	var length [8]byte
	binary.LittleEndian.PutUint64(length[:], uint64(len(raw)))
	h.Write(length[:])
	h.Write(raw)
}

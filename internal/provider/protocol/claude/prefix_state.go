// Purpose: Save and restore request-local SHA-256 prefix state without replaying cached signed blocks.
// Pattern: Immutable checkpoint of the standard-library hash state.
// Usage: The native request encoder keeps checkpoints only for its bounded unchanged history prefix.
// Related: internal/provider/protocol/claude/prefix.go, internal/provider/protocol/cloud_history.go.
package claude

import "encoding"

// PrefixState is an opaque in-memory checkpoint, never a persisted or network format.
type PrefixState struct {
	body []byte
	role string
}

// State snapshots the SHA-256 state promised by crypto/sha256.New's binary contract.
func (p *Prefix) State() (PrefixState, error) {
	body, err := p.digest.(encoding.BinaryMarshaler).MarshalBinary()
	return PrefixState{body: body, role: p.role}, err
}

// Restore replaces this request's running digest without mutating the checkpoint.
func (p *Prefix) Restore(state PrefixState) error {
	if err := p.digest.(encoding.BinaryUnmarshaler).UnmarshalBinary(state.body); err != nil {
		return err
	}
	p.role = state.role
	return nil
}

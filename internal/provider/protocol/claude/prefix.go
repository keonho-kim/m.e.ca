// Purpose: Fingerprint Claude's normalized earlier conversation without reserializing every prefix.
// Pattern: Request-owned incremental digest over unambiguous role and block boundaries.
// Usage: The request builder checks each continuation before appending its current blocks.
// Related: internal/provider/protocol/claude_request.go, internal/provider/protocol/claude/continuation.go.
package claude

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"hash"
)

type Prefix struct {
	digest hash.Hash
	role   string
	frame  [9]byte
}

func NewPrefix(system string, tools []Tool) (*Prefix, error) {
	if len(tools) == 0 {
		tools = nil
	}
	encoded, err := json.Marshal(tools)
	if err != nil {
		return nil, err
	}
	p := &Prefix{digest: sha256.New()}
	_, _ = p.digest.Write([]byte("meca-claude-prefix-v1\x00"))
	p.part(1, []byte(system))
	p.part(2, encoded)
	return p, nil
}

func (p *Prefix) part(kind byte, body []byte) {
	p.frame[0] = kind
	binary.LittleEndian.PutUint64(p.frame[1:], uint64(len(body)))
	_, _ = p.digest.Write(p.frame[:])
	_, _ = p.digest.Write(body)
}

func (p *Prefix) Append(role string, blocks []Block) error {
	if len(blocks) == 0 {
		return nil
	}
	if role != p.role {
		p.part(3, []byte(role))
		p.role = role
	}
	for _, block := range blocks {
		body, err := json.Marshal(block)
		if err != nil {
			return err
		}
		p.part(4, body)
	}
	return nil
}

func (p *Prefix) Value() string { return hex.EncodeToString(p.digest.Sum(nil)) }

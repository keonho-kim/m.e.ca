// Purpose: Reuse owned text and image groups before provider conversion without copying large keys.
// Pattern: Execution-owned immutable encoding cache with content fingerprints.
// Usage: The scoped HTTP generator calls Encoder.Messages before tokenization and Encode.
// Related: internal/provider/protocol/encoded_history.go, internal/provider/httpclient/request.go.
package protocol

import (
	"crypto/sha256"
	"encoding/binary"
	"hash"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

type sourceIdentity struct {
	role, content, reasoning, toolID, toolName string
	incomplete                                 bool
	calls, state                               [32]byte
	images                                     [32]byte
	group                                      *sourceGroup
	ordinal                                    int
}

type sourceGroup struct {
	keys        []sourceIdentity
	from, count int
	images      []bool
	roles       []string
}

type wireIdentity struct {
	role, content, thinking, reasoning, reasoningContent, toolID, toolName string
	raw                                                                    [32]byte
}

// Messages returns DTOs for this encoder's next request. Matching DTOs carry
// their immutable wire bytes; they must be consumed by Encode or TokenizerBody.
func (e *Encoder) Messages(kind connection.ProviderKind, model string, source []modelapi.Message) ([]Message, error) {
	size := 0
	for _, m := range source {
		size += 512 + len(m.Content) + len(m.Reasoning) + len(m.ReasoningState)
		for _, image := range m.Images {
			size += len(image.Data)*4/3 + 1024
		}
		for _, call := range m.Calls {
			size += len(call.Arguments) + len(call.Name) + len(call.ID)
		}
	}
	if e == nil || size < cacheThreshold || size > cacheCapacity {
		if e != nil {
			e.Clear()
		}
		return Messages(kind, source)
	}
	if e.kind != kind || e.model != model {
		e.Clear()
	}
	e.kind, e.model = kind, model
	out := make([]Message, 0, max(len(source), len(e.history)))
	previous := e.groups
	e.groups = e.groups[:0]
	for i := 0; i < len(source); {
		end := i + 1
		if source[i].Role == "tool" {
			for end < len(source) && source[end].Role == "tool" {
				end++
			}
		}
		groupIndex := len(e.groups)
		var group *sourceGroup
		if groupIndex < len(previous) {
			p := previous[groupIndex]
			if p.matches(source[i:end]) && p.from+p.count <= len(e.history) && e.history[p.from].source.group == p {
				group = p
			}
		}
		if group != nil {
			entries := e.history[group.from : group.from+group.count]
			group.from = len(out)
			for j, entry := range entries {
				key := sourceIdentity{group: group, ordinal: j}
				out = append(out, Message{Role: group.roles[j], Content: "", encoded: entry.json, source: &key, sourceBytes: 512 + len(entry.json), hasImages: group.images[j]})
			}
			e.groups = append(e.groups, group)
			i = end
			continue
		}
		converted, err := Messages(kind, source[i:end])
		if err != nil {
			e.Clear()
			return nil, err
		}
		keys := make([]sourceIdentity, end-i)
		for at := i; at < end; at++ {
			keys[at-i] = identifySource(source[at])
		}
		group = &sourceGroup{keys: keys, from: len(out), count: len(converted), images: make([]bool, len(converted)), roles: make([]string, len(converted))}
		for j, message := range converted {
			key := sourceIdentity{group: group, ordinal: j}
			message.source = &key
			group.images[j] = message.HasImages()
			group.roles[j] = message.Role
			out = append(out, message)
		}
		e.groups = append(e.groups, group)
		i = end
	}
	if len(previous) > len(e.groups) {
		clear(previous[len(e.groups):])
	}
	return out, nil
}

func (g *sourceGroup) matches(source []modelapi.Message) bool {
	if len(g.keys) != len(source) {
		return false
	}
	for i, m := range source {
		if identifySource(m) != g.keys[i] {
			return false
		}
	}
	return true
}

func identifySource(m modelapi.Message) sourceIdentity {
	calls, state := m.NativeFingerprint()
	key := sourceIdentity{role: m.Role, content: m.Content, reasoning: m.Reasoning, toolID: m.ToolCallID, toolName: m.ToolName, incomplete: m.Incomplete, calls: calls, state: state}
	if len(m.Images) > 0 {
		h := sha256.New()
		for _, image := range m.Images {
			fingerprintPart(h, []byte(image.Digest))
			fingerprintPart(h, []byte(image.Path))
			fingerprintPart(h, []byte(image.MIME))
			fingerprintPart(h, []byte(image.Format))
			var dimensions [32]byte
			binary.LittleEndian.PutUint64(dimensions[:8], uint64(image.Width))
			binary.LittleEndian.PutUint64(dimensions[8:16], uint64(image.Height))
			binary.LittleEndian.PutUint64(dimensions[16:24], uint64(len(image.Data)))
			if image.Representative {
				dimensions[24] = 1
			}
			fingerprintPart(h, dimensions[:])
		}
		copy(key.images[:], h.Sum(nil))
	}
	return key
}

func identifyWire(m Message) wireIdentity {
	content, _ := m.Content.(string)
	key := wireIdentity{role: m.Role, content: content, thinking: m.Thinking, reasoning: m.Reasoning, reasoningContent: m.ReasoningContent, toolID: m.ToolCallID, toolName: m.ToolName}
	if len(m.Calls) > 0 || len(m.ReasoningDetails) > 0 {
		h := sha256.New()
		fingerprintPart(h, m.Calls)
		for _, raw := range m.ReasoningDetails {
			fingerprintPart(h, raw)
		}
		copy(key.raw[:], h.Sum(nil))
	}
	return key
}

func fingerprintPart(h hash.Hash, raw []byte) {
	var size [8]byte
	binary.LittleEndian.PutUint64(size[:], uint64(len(raw)))
	h.Write(size[:])
	h.Write(raw)
}

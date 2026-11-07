// Purpose: Reuse a bounded unchanged native history prefix as replayable JSON fragments.
// Pattern: Execution-owned prefix cache with request-local signature and call-identity state.
// Usage: Encoder dispatches Claude and Gemini here; Clear releases all retained history.
// Related: internal/provider/protocol/claude_cloud.go, internal/provider/protocol/gemini_cloud.go.
package protocol

import (
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
	"github.com/keonho-kim/meca/internal/provider/protocol/claude"
)

type cloudCallID struct{ local, wire string }
type cloudEntry struct {
	source *cloudSourceID
	key    wireIdentity
	role   string
	parts  []byte
	prefix claude.PrefixState
	ids    []cloudCallID
}

type cloudRequest struct {
	endpoint, field, initial string
	header                   []byte
	prefix                   *claude.Prefix
	wireIDs                  map[string]string
}

func (e *Encoder) encodeCloud(kind connection.ProviderKind, base string, request modelapi.Request, messages []Message, controls Controls) (string, Body, error) {
	if e == nil {
		return singleBody(kind, base, nil, request, messages, controls)
	}
	if e.kind != kind || e.model != request.ModelID {
		e.Clear()
	}
	e.kind, e.model = kind, request.ModelID
	if len(messages) == 0 || messages[0].cloudSource == nil {
		e.cloudGroups = nil
		e.cloudSourceBytes = 0
	}
	size, text := textHistorySize(messages)
	for _, message := range messages {
		if message.Role != "system" && message.Role != "user" && message.Role != "assistant" && message.Role != "tool" {
			text = false
			break
		}
	}
	if !text || size < cacheThreshold {
		e.cloud, e.cloudInitial, e.cloudRetained = nil, "", 0
		return singleBody(kind, base, nil, request, messages, controls)
	}
	var native cloudRequest
	var err error
	if kind == connection.Claude {
		native, err = newClaudeCloud(base, request, messages, controls)
	} else {
		native, err = newGeminiCloud(base, request, messages, controls)
	}
	if err != nil {
		e.Clear()
		return "", Body{}, err
	}
	previous := e.cloud
	unchanged := 0
	if e.cloudInitial == native.initial {
		for unchanged < min(len(previous), len(messages)) && previous[unchanged].matches(messages[unchanged]) {
			unchanged++
		}
	}
	e.cloud, e.cloudInitial, e.cloudRetained = e.cloud[:0], native.initial, 0
	header := requestHeader{controls: native.header}
	parts, bytes := header.appendTo(make([][]byte, 0, 4*len(messages)+4), native.field)
	sequence := cloudSequence{parts: parts, size: bytes, field: "content"}
	if kind == connection.Gemini {
		sequence.field = "parts"
	}
	for index, message := range messages {
		var entry cloudEntry
		if index < unchanged {
			entry = previous[index]
			if native.prefix != nil {
				err = native.prefix.Restore(entry.prefix)
			} else {
				for _, id := range entry.ids {
					native.wireIDs[id.local] = id.wire
				}
			}
		} else if kind == connection.Claude {
			entry, err = native.claudeEntry(message)
		} else {
			entry, err = native.geminiEntry(message)
		}
		if err != nil {
			e.Clear()
			return "", Body{}, err
		}
		if err = sequence.add(entry.role, entry.parts); err != nil {
			e.Clear()
			return "", Body{}, err
		}
		if index == len(e.cloud) {
			if index >= unchanged {
				entry.source = message.cloudSource
				if entry.source == nil {
					entry.key = identifyWire(message)
				}
			}
			weight := entry.retainedBytes()
			if e.cloudRetained+weight <= e.limit()-e.cloudSourceBytes {
				if native.prefix != nil && index >= unchanged {
					state, stateErr := native.prefix.State()
					if stateErr != nil {
						continue
					} // An unavailable hash snapshot only disables retention.
					entry.prefix = state
				}
				e.cloud = append(e.cloud, entry)
				e.cloudRetained += weight
			}
		}
	}
	if len(previous) > len(e.cloud) {
		clear(previous[len(e.cloud):])
	}
	e.shrinkCloud()
	body := sequence.finish()
	if native.prefix != nil {
		body.bind = claudeContinuation(native.prefix.Value())
	}
	return native.endpoint, body, nil
}

func (e *Encoder) shrinkCloud() {
	if len(e.cloud) == 0 {
		e.cloud = nil
	} else if cap(e.cloud) > max(16, 2*len(e.cloud)) {
		e.cloud = append([]cloudEntry(nil), e.cloud...)
	}
}

func (entry cloudEntry) matches(message Message) bool {
	if message.cloudSource != nil {
		return entry.source == message.cloudSource
	}
	return entry.source == nil && entry.key == identifyWire(message)
}

func (entry cloudEntry) retainedBytes() int {
	key := entry.key
	bytes := 1024 + len(entry.parts) + len(key.content) + len(key.thinking) + len(key.reasoning) + len(key.reasoningContent) + len(key.toolID) + len(key.toolName)
	for _, id := range entry.ids {
		bytes += 32 + len(id.local) + len(id.wire)
	}
	return bytes
}

type cloudSequence struct {
	parts          [][]byte
	size           int
	field, role    string
	open, hasParts bool
}

func (s *cloudSequence) append(parts ...[]byte) {
	s.parts = append(s.parts, parts...)
	for _, part := range parts {
		s.size += len(part)
	}
}

func (s *cloudSequence) add(role string, parts []byte) error {
	if len(parts) == 0 {
		return nil
	}
	if !s.open || s.role != role {
		if s.open {
			s.append([]byte(`]},`))
		} else {
			s.append([]byte(`[`))
		}
		encoded, err := wireJSON.Marshal(role)
		if err != nil {
			return err
		}
		s.append([]byte(`{"role":`), encoded, []byte(`,"`+s.field+`":[`))
		s.role, s.open, s.hasParts = role, true, false
	}
	if s.hasParts {
		s.append([]byte(`,`))
	}
	s.append(parts)
	s.hasParts = true
	return nil
}

func (s *cloudSequence) finish() Body {
	if s.open {
		s.append([]byte(`]}]}`))
	} else {
		s.append([]byte(`null}`))
	}
	return Body{parts: s.parts, size: s.size}
}

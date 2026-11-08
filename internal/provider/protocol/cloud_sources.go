// Purpose: Reuse validated native source groups without repeatedly parsing unchanged tool argument objects.
// Pattern: Execution-owned immutable DTO snapshots under half the shared encoding budget.
// Usage: Encoder.Messages checks source fingerprints; native wire fragments use the remaining budget.
// Related: internal/provider/protocol/cloud_history.go, internal/provider/protocol/source_history.go.
package protocol

import (
	"slices"

	"github.com/keonho-kim/meca/internal/provider/protocol/native"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/provider/connection"
)

// A separate nonzero-size identity never pins a source group or argument buffer
// through a wire cache entry. Returned prepared DTOs are immutable, like other Encoder.Messages values.
type cloudSourceID byte

type cloudSourceGroup struct {
	keys     []sourceIdentity
	messages []Message
	bytes    int
}

func (e *Encoder) cloudMessages(kind connection.ProviderKind, model string, source []modelapi.Message) ([]Message, error) {
	if e == nil {
		return Messages(kind, source)
	}
	if e.kind != kind || e.model != model {
		e.Clear()
	}
	e.kind, e.model = kind, model
	if sourceHistoryBytes(source) < cacheThreshold {
		e.Clear()
		e.kind, e.model = kind, model
		return Messages(kind, source)
	}
	previous := e.cloudGroups
	e.cloudGroups, e.cloudSourceBytes = e.cloudGroups[:0], 0
	out := make([]Message, 0, len(source))
	for index, groupIndex := 0, 0; index < len(source); groupIndex++ {
		end := index + 1
		if source[index].Role == "tool" {
			for end < len(source) && source[end].Role == "tool" {
				end++
			}
		}
		var group *cloudSourceGroup
		if groupIndex < len(previous) && sameSource(previous[groupIndex].keys, source[index:end]) {
			group = previous[groupIndex]
		}
		var converted []Message
		if group != nil {
			converted = group.messages
		} else {
			var err error
			converted, err = Messages(kind, source[index:end])
			if err != nil {
				e.Clear()
				return nil, err
			}
		}
		weight := cloudGroupBytes(source[index:end], converted)
		if groupIndex == len(e.cloudGroups) && e.cloudSourceBytes+weight <= e.limit()/2 {
			if group == nil {
				group = &cloudSourceGroup{keys: make([]sourceIdentity, end-index), messages: snapshotCloudMessages(converted), bytes: weight}
				for at := index; at < end; at++ {
					group.keys[at-index] = identifySource(source[at])
				}
				for at := range group.messages {
					group.messages[at].cloudSource = new(cloudSourceID)
				}
			}
			e.cloudGroups = append(e.cloudGroups, group)
			e.cloudSourceBytes += group.bytes
			converted = group.messages
		}
		for offset, message := range converted {
			if offset < end-index && source[index+offset].Role == "assistant" {
				current := source[index+offset]
				message.nativeCalls = current.Calls
				if !current.Incomplete {
					message.continuation = current.ReasoningState
				}
			}
			out = append(out, message)
		}
		index = end
	}
	if len(previous) > len(e.cloudGroups) {
		clear(previous[len(e.cloudGroups):])
	}
	if len(e.cloudGroups) == 0 {
		e.cloudGroups = nil
	} else if cap(e.cloudGroups) > max(16, 2*len(e.cloudGroups)) {
		e.cloudGroups = append([]*cloudSourceGroup(nil), e.cloudGroups...)
	}
	for len(e.cloud) > 0 && e.cloudRetained > e.limit()-e.cloudSourceBytes {
		last := len(e.cloud) - 1
		e.cloudRetained -= e.cloud[last].retainedBytes()
		e.cloud[last] = cloudEntry{}
		e.cloud = e.cloud[:last]
	}
	e.shrinkCloud()
	return out, nil
}

func snapshotCloudMessages(source []Message) []Message {
	out := slices.Clone(source)
	for index := range out {
		message := &out[index]
		// Source fingerprints prove validation; the current caller rebinds these
		// immutable originals. Never retain a second argument/continuation body.
		message.Calls, message.continuation, message.nativeCalls = nil, nil, nil
		if parts, ok := message.Content.([]native.ContentPart); ok {
			owned := slices.Clone(parts)
			for at := range owned {
				if owned[at].ImageURL != nil {
					value := *owned[at].ImageURL
					owned[at].ImageURL = &value
				}
			}
			message.Content = owned
		}
	}
	return out
}

func cloudGroupBytes(source []modelapi.Message, messages []Message) int {
	weight := 1024 + 512*(len(source)+len(messages))
	for _, message := range source {
		weight += len(message.Role) + len(message.Content) + len(message.Reasoning) + len(message.ToolCallID) + len(message.ToolName)
	}
	for _, message := range messages {
		if parts, ok := message.Content.([]native.ContentPart); ok {
			for _, part := range parts {
				if part.ImageURL != nil {
					weight += len(part.ImageURL.URL)
				}
			}
		}
	}
	return weight
}

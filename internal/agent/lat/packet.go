// Purpose: Prepare one bounded LAT envelope and add independently selected evidence once.
// Pattern: Immutable phase-local preparation with a pure final assembly.
// Usage: The runtime prepares a frame, selects SourceBudget bytes, then builds its input.
// Related: internal/agent/lat/memory/tail.go, internal/agent/run/lat_context.go.
package lat

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode/utf8"

	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type Input struct {
	Prompt, Instructions, Root, Objective, Draft string
	Plan, Critique                               string
	Context                                      json.RawMessage
	OutputReserve                                int
	ContextWindow, MaxOutput                     int
}

// Packet is the bounded user input and reserved generation budget for a native LAT phase.
type Packet struct {
	Content   string
	MaxOutput int
}

// Frame holds one immutable encoded envelope for the lifetime of a phase.
type Frame struct {
	fixed          []byte
	budget, output int
}

type reviewPacket struct {
	Plan     string          `json:"proposed_plan,omitempty"`
	Critique string          `json:"plan_critique,omitempty"`
	Root     string          `json:"project_root"`
	Request  string          `json:"latest_user_request"`
	Context  json.RawMessage `json:"independent_evidence,omitempty"`
	Draft    string          `json:"answer_draft_not_proof_of_effects"`
}

const evidenceField = `,"independent_evidence":`

// Prepare validates and encodes mandatory requirements once for one native phase.
func Prepare(in Input) (Frame, error) {
	if err := prompt.CheckTemplate(in.Prompt); err != nil {
		return Frame{}, err
	}
	// Shared instructions are already assembled from individually checked files.
	// Their combined length is governed by the effective context, not a file cap.
	if !utf8.ValidString(in.Instructions) || strings.ContainsRune(in.Instructions, 0) {
		return Frame{}, fmt.Errorf("LAT shared instructions have invalid encoding")
	}
	systemBytes := len(in.Prompt) + len(in.Instructions) + 2
	window := in.ContextWindow
	if window <= 0 {
		window = 262144
	}
	output := in.MaxOutput
	if output == 0 {
		output = 4096
	}
	if in.OutputReserve > 0 {
		output = in.OutputReserve
	} else {
		output = min(output, window/4)
	}
	capacity := min(latmemory.MaxBytes, window-output-systemBytes-1024)
	packet := reviewPacket{Root: in.Root, Request: in.Objective, Plan: in.Plan, Critique: in.Critique}
	fixed, err := json.Marshal(packet)
	if err != nil || output < 1 || len(fixed)+1024 > capacity {
		return Frame{}, fmt.Errorf("LAT current request and instructions cannot fit the available context; original retained")
	}
	budget := min(capacity, max(64*1024, len(fixed)+32*1024))
	packet.Draft = latmemory.Excerpt(in.Draft, min(12*1024, (budget-len(fixed))/5))
	for {
		fixed, err = json.Marshal(packet)
		if err != nil {
			return Frame{}, err
		}
		if len(fixed)+1024 <= budget {
			return Frame{fixed: fixed, budget: budget, output: output}, nil
		}
		if packet.Draft == "" {
			return Frame{}, fmt.Errorf("LAT mandatory input cannot fit available context")
		}
		if len(packet.Draft) < 80 {
			packet.Draft = ""
		} else {
			packet.Draft = latmemory.Excerpt(packet.Draft, len(packet.Draft)/2)
		}
	}
}

// SourceBudget returns the remaining evidence bytes after the encoded envelope.
func (frame Frame) SourceBudget() int { return frame.budget - len(frame.fixed) - len(evidenceField) }

// Build combines valid evidence with the immutable envelope in one final buffer.
func (frame Frame) Build(evidence json.RawMessage) (Packet, error) {
	if len(frame.fixed) == 0 {
		return Packet{}, fmt.Errorf("LAT phase frame is uninitialized")
	}
	if len(evidence) > frame.SourceBudget() {
		return Packet{}, fmt.Errorf("LAT independent evidence exceeds available context; no source was silently omitted")
	}
	if !json.Valid(evidence) {
		return Packet{}, fmt.Errorf("LAT independent source packet is unavailable")
	}
	// Both fragments are valid immutable JSON. Insert the evidence field without
	// reparsing or re-escaping all requirements, planning text and observations.
	var body strings.Builder
	body.Grow(len(frame.fixed) + len(evidenceField) + len(evidence))
	body.Write(frame.fixed[:len(frame.fixed)-1])
	body.WriteString(evidenceField)
	body.Write(evidence)
	body.WriteByte('}')
	return Packet{Content: body.String(), MaxOutput: frame.output}, nil
}

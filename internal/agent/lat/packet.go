// Purpose: Assemble a final review from independent memory, original deltas and the current request.
// Pattern: Pure context builder with mandatory requirements and bounded answer excerpts.
// Usage: The runtime obtains SourceBudget before loading a complete independent source tail.
// Related: internal/agent/lat/memory/tail.go, internal/agent/run/lat_context.go.
package lat

import (
	"encoding/json"
	"fmt"
	latmemory "github.com/keonho-kim/meca/internal/agent/lat/memory"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	"github.com/keonho-kim/meca/internal/agent/prompt"
)

type Input struct {
	Prompt, Instructions, Root, Objective, Draft string
	Context                                      json.RawMessage
	ContextWindow, MaxOutput                     int
}

type reviewPacket struct {
	Root    string          `json:"project_root"`
	Request string          `json:"latest_user_request"`
	Context json.RawMessage `json:"independent_evidence"`
	Draft   string          `json:"answer_draft_not_proof_of_effects"`
}

func frame(in Input) (reviewPacket, string, int, int, error) {
	system, err := prompt.Compose(in.Prompt, in.Instructions)
	if err != nil {
		return reviewPacket{}, "", 0, 0, err
	}
	window := in.ContextWindow
	if window <= 0 {
		window = 262144
	}
	output := in.MaxOutput
	if output == 0 {
		output = 4096
	}
	output = min(output, window/4)
	capacity := min(latmemory.MaxBytes, window-output-len(system)-1024)
	packet := reviewPacket{Root: in.Root, Request: in.Objective}
	fixed, err := json.Marshal(packet)
	if err != nil || output < 1 || len(fixed)+1024 > capacity {
		return packet, system, 0, 0, fmt.Errorf("LAT current request and instructions cannot fit the available context; original retained")
	}
	budget := min(capacity, max(64*1024, len(fixed)+32*1024))
	packet.Draft = latmemory.Excerpt(in.Draft, min(12*1024, (budget-len(fixed))/5))
	for {
		encoded, err := json.Marshal(packet)
		if err != nil {
			return packet, system, 0, 0, err
		}
		if len(encoded)+1024 <= budget {
			break
		}
		if packet.Draft == "" {
			return packet, system, 0, 0, fmt.Errorf("LAT mandatory input cannot fit available context")
		}
		if len(packet.Draft) < 80 {
			packet.Draft = ""
		} else {
			packet.Draft = latmemory.Excerpt(packet.Draft, len(packet.Draft)/2)
		}
	}
	return packet, system, budget, output, nil
}

func SourceBudget(in Input) (int, error) {
	packet, _, budget, _, err := frame(in)
	if err != nil {
		return 0, err
	}
	fixed, err := json.Marshal(packet)
	if err != nil {
		return 0, err
	}
	return budget - len(fixed) - 512, nil
}

func Build(in Input) (modelapi.Request, error) {
	packet, system, budget, output, err := frame(in)
	if err != nil {
		return modelapi.Request{}, err
	}
	if !json.Valid(in.Context) {
		return modelapi.Request{}, fmt.Errorf("LAT independent source packet is unavailable")
	}
	packet.Context = in.Context
	body, err := json.Marshal(packet)
	if err != nil || len(body) > budget {
		return modelapi.Request{}, fmt.Errorf("LAT independent evidence exceeds available context; no source was silently omitted")
	}
	return modelapi.Request{MaxOutput: output, Messages: []modelapi.Message{{Role: "system", Content: system}, {Role: "user", Content: string(body)}}}, nil
}

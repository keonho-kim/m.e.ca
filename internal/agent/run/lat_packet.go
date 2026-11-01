// Purpose: Fit a LAT phase input to its role budget and combine independently selected evidence.
// Pattern: Phase preparation use case.
// Usage: runLATPhase supplies its fresh read-only tools before constructing the child runner.
// Related: internal/agent/run/lat_fork.go, internal/agent/run/lat_context.go, internal/agent/lat/packet.go.
package run

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/keonho-kim/meca/internal/agent/inference"
	"github.com/keonho-kim/meca/internal/agent/lat"
	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

func (s *execution) prepareLATPacket(ctx context.Context, instructions string, role inference.Role, input lat.Input, tools toolapi.Executor) (lat.Packet, []toolapi.Definition, error) {
	parameters, err := inference.Resolve(s.runner.Inference, role)
	if err != nil {
		return lat.Packet{}, nil, err
	}
	// Account for shared instructions and tool schemas before selecting evidence.
	definitions := imageDefinitions(tools.Definitions(), s.runner.ImageInput)
	encodedDefinitions, err := json.Marshal(definitions)
	if err != nil {
		return lat.Packet{}, nil, err
	}
	window := s.latWindow()
	if window <= 0 {
		window = 262144
	}
	if role == inference.Agent {
		input.OutputReserve, err = modelapi.OutputBudget(parameters.MaxOutputTokens, window, 0, 0)
		if err != nil {
			return lat.Packet{}, nil, err
		}
	}
	window -= len(encodedDefinitions) + 4096
	if window <= 0 {
		return lat.Packet{}, nil, fmt.Errorf("LAT instructions and tools exceed the available context")
	}
	input.Prompt, input.Instructions = instructions, s.system
	input.Root, input.Objective = s.runner.ProjectRoot, s.checkpoint.Objective
	input.ContextWindow, input.MaxOutput = window, parameters.MaxOutputTokens
	frame, err := lat.Prepare(input)
	if err != nil {
		return lat.Packet{}, nil, err
	}
	input.Context, err = s.prepareLATContext(ctx, frame.SourceBudget())
	if err != nil {
		return lat.Packet{}, nil, err
	}
	packet, err := frame.Build(input.Context)
	if err != nil {
		return lat.Packet{}, nil, err
	}

	return packet, definitions, nil
}

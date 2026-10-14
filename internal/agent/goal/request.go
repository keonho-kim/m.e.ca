// Purpose: Preserve an undecomposed user request when model-generated Goal analysis is unavailable.
// Pattern: Simple Module.
// Usage: Goal orchestration uses Unsplit after bounded format correction fails.
// Related: internal/agent/goal/state.go, internal/agent/run/goal_model.go.
package goal

func Unsplit(id, request string) (State, error) {
	objective := request
	if len(objective) > 4096 {
		objective = "Handle the original user request in full, preserving its constraints."
	}
	state := State{
		ID: id, Request: request, Objective: objective, Status: "pending",
		Tasks: []Task{{ID: id + "_task_1", Objective: objective, Kind: "request", Status: "pending"}},
	}
	return state, state.Check()
}

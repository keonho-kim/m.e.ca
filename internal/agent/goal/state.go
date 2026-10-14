// Purpose: Define internal Goal decomposition and evidence-based sequential Task states.
// Pattern: Goal state machine and plain-text parser.
// Usage: Goal orchestration owns identifiers and transitions; none are terminal UI commands.
// Related: internal/agent/run/goal.go, internal/storage/sqlite/monitor.go.
package goal

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Assignment struct{ GoalID, TaskID, GoalObjective, TaskObjective, Kind string }
type Task struct{ ID, Objective, Kind, Status, Result, Evidence string }
type State struct {
	ID, Request, Objective, Status string
	Tasks                          []Task
}

func Parse(text, id, request string) (State, error) {
	state := State{ID: id, Request: request, Status: "pending"}
	if len(text) > 16*1024 || !utf8.ValidString(text) {
		return state, fmt.Errorf("invalid Goal analysis size or encoding")
	}
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	if len(lines) > 9 {
		return state, fmt.Errorf("analysis requires at most 8 Tasks")
	}
	if len(lines) < 2 || !strings.HasPrefix(lines[0], "GOAL: ") {
		return state, fmt.Errorf("Goal analysis requires GOAL: and TASK work: or TASK answer: lines")
	}
	state.Objective = strings.TrimSpace(strings.TrimPrefix(lines[0], "GOAL: "))
	for _, line := range lines[1:] {
		kind, objective, ok := strings.Cut(line, ": ")
		if !ok || kind != "TASK work" && kind != "TASK answer" {
			return state, fmt.Errorf("invalid Task line")
		}
		state.Tasks = append(state.Tasks, Task{ID: fmt.Sprintf("%s_task_%d", id, len(state.Tasks)+1), Kind: strings.TrimPrefix(kind, "TASK "), Objective: strings.TrimSpace(objective), Status: "pending"})
	}
	return state, state.Check()
}
func (s State) Check() error {
	if s.ID == "" || len(s.ID) > 96 || !utf8.ValidString(s.ID+s.Request+s.Objective) || len(s.Request) > 64*1024 || s.Objective == "" || len(s.Objective) > 16*1024 || len(s.Tasks) < 1 || len(s.Tasks) > 50 {
		return fmt.Errorf("invalid Goal")
	}
	switch s.Status {
	case "pending", "running", "completed", "blocked", "stopped", "interrupted":
	default:
		return fmt.Errorf("invalid Goal status")
	}
	seen := make(map[string]bool)
	for _, task := range s.Tasks {
		if task.ID == "" || len(task.ID) > 128 || task.Objective == "" || len(task.Objective) > 4096 || task.Kind != "work" && task.Kind != "answer" || len(task.Result) > 16384 || len(task.Evidence) > 4096 {
			return fmt.Errorf("invalid Task")
		}
		if seen[task.ID] || !utf8.ValidString(task.Objective+task.Result+task.Evidence) {
			return fmt.Errorf("duplicate or malformed Task")
		}
		seen[task.ID] = true
		if task.Status == "completed" && task.Kind == "work" && strings.TrimSpace(task.Evidence) == "" {
			return fmt.Errorf("completed work has no evidence")
		}
		switch task.Status {
		case "pending", "running", "completed", "blocked", "stopped", "interrupted":
		default:
			return fmt.Errorf("invalid Task status")
		}
	}
	if s.Status == "completed" {
		for _, task := range s.Tasks {
			if task.Status != "completed" {
				return fmt.Errorf("Goal has unfinished Tasks")
			}
		}
	}
	return nil
}
func (s *State) Start(index int) error {
	if (s.Status != "pending" && s.Status != "running") || index < 0 || index >= len(s.Tasks) || s.Tasks[index].Status != "pending" {
		return fmt.Errorf("Task cannot start")
	}
	for _, task := range s.Tasks[:index] {
		if task.Status != "completed" {
			return fmt.Errorf("previous Task is unfinished")
		}
	}
	s.Status = "running"
	s.Tasks[index].Status = "running"
	return nil
}
func (s *State) Finish(index int, status, text, evidence string) error {
	if index < 0 || index >= len(s.Tasks) || s.Tasks[index].Status != "running" {
		return fmt.Errorf("Task is not running")
	}
	task := &s.Tasks[index]
	task.Result, task.Evidence = text, evidence
	switch {
	case status == "verified" && evidence != "", task.Kind == "answer" && status == "answered":
		task.Status = "completed"
	case status == "interrupted":
		task.Status = "interrupted"
		s.Status = "interrupted"
	case status == "needs_input":
		task.Status = "blocked"
		s.Status = "blocked"
	default:
		task.Status = "stopped"
		s.Status = "stopped"
	}
	if task.Status == "completed" && index == len(s.Tasks)-1 {
		s.Status = "completed"
	}
	return s.Check()
}

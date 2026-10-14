// Purpose: Parse and bound model-generated Plan proposals.
// Pattern: Boundary parser.
// Usage: Called by planning requests and Plan transcript projection.
// Related: internal/agent/plan/plan.go, internal/agent/run/planning.go.
package plan

import (
	"encoding/json"
	"fmt"
	"strings"
)

type PlanProposal struct {
	Objective string   `json:"objective"`
	Steps     []string `json:"steps"`
}

func ParsePlanProposal(content string) (PlanProposal, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```json\n") && strings.HasSuffix(content, "\n```") {
		content = strings.TrimSuffix(strings.TrimPrefix(content, "```json\n"), "\n```")
	}
	if len(content) > 64*1024 {
		return PlanProposal{}, fmt.Errorf("Plan proposal exceeds size limit")
	}
	var proposal PlanProposal
	if err := json.Unmarshal([]byte(content), &proposal); err != nil {
		return proposal, fmt.Errorf("invalid structured Plan: %w", err)
	}
	if err := checkProposal(proposal); err != nil {
		return proposal, err
	}
	return proposal, nil
}

func checkProposal(proposal PlanProposal) error {
	if strings.TrimSpace(proposal.Objective) == "" || len(proposal.Objective) > 16*1024 || len(proposal.Steps) == 0 || len(proposal.Steps) > 50 {
		return fmt.Errorf("Plan requires a bounded objective and 1–50 steps")
	}
	for _, description := range proposal.Steps {
		if strings.TrimSpace(description) == "" || len(description) > 4096 {
			return fmt.Errorf("invalid Plan step description")
		}
	}
	return nil
}

// Purpose: Resolve explicit or automatic output budgets within known model capacity.
// Pattern: Pure Function.
// Usage: Inference validates configuration; providers supply effective serving limits.
// Related: internal/agent/inference/settings.go, internal/provider/httpclient/context.go.
package model

import "fmt"

// CheckOutputBudget accepts zero for automatic output and positive explicit limits.
func CheckOutputBudget(tokens int) error {
	if tokens < 0 || tokens > 1<<30 {
		return fmt.Errorf("max_output_tokens must be between 0 (automatic) and 1073741824")
	}
	return nil
}

// OutputBudget uses 75% of context for automatic output. Input is an exact count,
// or zero when unavailable; unknown input is never estimated here.
func OutputBudget(requested, capacity, serverOutput, input int) (int, error) {
	if err := CheckOutputBudget(requested); err != nil {
		return 0, err
	}
	if capacity < 1 || capacity > 1<<30 || input < 0 || serverOutput < 0 {
		return 0, fmt.Errorf("invalid model capacity")
	}
	if input >= capacity {
		return 0, fmt.Errorf("model context exhausted: input %d tokens, capacity %d; history was preserved", input, capacity)
	}
	output := requested
	if output == 0 {
		output = capacity/4*3 + capacity%4*3/4
	}
	if serverOutput > 0 {
		output = min(output, serverOutput)
	}
	output = min(output, capacity-max(1, input))
	if output < 1 {
		return 0, fmt.Errorf("model context has no output capacity")
	}
	return output, nil
}

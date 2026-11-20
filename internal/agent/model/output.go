// Purpose: Resolve explicit or automatic output budgets within known model capacity.
// Pattern: Pure Function.
// Usage: Inference validates configuration; providers supply effective serving limits.
// Related: internal/agent/inference/settings.go, internal/provider/httpclient/context.go.
package model

import (
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

// CheckOutputBudget accepts zero for automatic output and positive explicit limits.
func CheckOutputBudget(tokens int) error {
	if tokens < 0 || tokens > 1<<30 {
		return faults.Errorf(diagcatalog.ModelMaxOutputTokensMustBeBetweenAutomatic)
	}
	return nil
}

// OutputBudget uses 75% of context for automatic output. Input is an exact count,
// or zero when unavailable; unknown input is never estimated here.
func OutputBudget(requested, capacity, serverOutput, input int) (int, error) {
	return Request{MaxOutput: requested}.OutputBudget(capacity, serverOutput, input)
}

// OutputBudget resolves the request's output limits against the effective context window.
func (r Request) OutputBudget(capacity, serverOutput, input int) (int, error) {
	if err := CheckOutputBudget(r.MaxOutput); err != nil {
		return 0, err
	}
	if capacity < 1 || capacity > 1<<30 || input < 0 || serverOutput < 0 {
		return 0, faults.Errorf(diagcatalog.ModelInvalidModelCapacity)
	}
	if input >= capacity {
		return 0, &ContextLimitError{Input: input, Capacity: capacity, Tokenized: true}
	}
	output := r.MaxOutput
	if output == 0 {
		output = capacity/4*3 + capacity%4*3/4
	}
	if r.OutputContextPercent > 0 {
		output = min(output, int(int64(capacity)*int64(r.OutputContextPercent)/100))
	}
	if serverOutput > 0 {
		output = min(output, serverOutput)
	}
	output = min(output, capacity-max(1, input))
	if output < 1 {
		return 0, faults.Errorf(diagcatalog.ModelModelContextHasNoOutputCapacity)
	}
	return output, nil
}

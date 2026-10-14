// Purpose: Validate an explicitly requested temperature without confusing zero with omission.
// Pattern: Simple Module.
// Usage: Role settings and provider request boundaries use CheckTemperature.
// Related: internal/agent/model/model.go.
package model

import (
	"fmt"
	"math"
)

func CheckTemperature(value *float64) error {
	if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > 2) {
		return fmt.Errorf("temperature must be a finite number between 0 and 2")
	}
	return nil
}

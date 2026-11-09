// Purpose: Validate an explicitly requested temperature without confusing zero with omission.
// Pattern: Simple Module.
// Usage: Role settings and provider request boundaries use CheckTemperature.
// Related: internal/agent/model/model.go.
package model

import (
	"math"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func CheckTemperature(value *float64) error {
	if value != nil && (math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 || *value > 2) {
		return faults.Errorf(diagcatalog.ModelTemperatureMustBeAFiniteNumberBetween)
	}
	return nil
}

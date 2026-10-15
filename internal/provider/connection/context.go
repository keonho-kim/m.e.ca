// Purpose: Define the configurable context budget and combine it with reported server capacity.
// Pattern: Pure limit policy.
// Usage: Config validates the global budget; generation resolves the smaller effective limit.
// Related: internal/config/settings.go, internal/provider/httpclient/request.go.
package connection

import "fmt"

const DefaultContextLength = 262144

func CheckContextLength(tokens int) error {
	if tokens < 1024 || tokens > 1<<30 {
		return fmt.Errorf("context_length must be between 1024 and 1073741824 tokens")
	}
	return nil
}

func EffectiveContextLength(configured, reported int) int {
	if reported > 0 {
		return min(configured, reported)
	}
	return configured
}

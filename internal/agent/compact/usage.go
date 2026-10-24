// Purpose: Define current-context compaction budgets and byte-based fallback estimates.
// Pattern: Simple Module.
// Usage: Configuration and request runtime use Settings and TokenCount.
// Related: internal/config/settings.go, internal/agent/compact/controller.go.
package compact

import "fmt"

const MaxBodyBytes = 16 * 1024 * 1024

const bytesPerToken int64 = 3

type Settings struct {
	KeepRecentTokens int   `toml:"keep_recent_tokens"`
	ReserveTokens    int64 `toml:"reserve_tokens"`
	MaxSummaryBytes  int   `toml:"max_summary_bytes"`
	MaxInputBytes    int   `toml:"max_input_bytes"`
}

func Defaults() Settings {
	return Settings{KeepRecentTokens: 20000, ReserveTokens: 16384, MaxSummaryBytes: MaxBodyBytes, MaxInputBytes: MaxBodyBytes}
}
func (s Settings) Check() error {
	if s.KeepRecentTokens < 0 || s.KeepRecentTokens > MaxBodyBytes || s.ReserveTokens < 1 || s.ReserveTokens > 1000000000 || s.MaxSummaryBytes < 1024 || s.MaxSummaryBytes > MaxBodyBytes || s.MaxInputBytes < 8192 || s.MaxInputBytes > MaxBodyBytes {
		return fmt.Errorf("invalid compaction settings")
	}
	return nil
}

type TokenCount struct {
	Amount    int64
	Estimated bool
	Kind      string
}

// EstimateBytes approximates mixed code and multilingual requests at three UTF-8
// bytes per token. It is only a compaction trigger, never displayed provider usage.
func EstimateBytes(bytes int64) TokenCount {
	return TokenCount{Amount: (bytes + bytesPerToken - 1) / bytesPerToken, Estimated: true, Kind: "request"}
}

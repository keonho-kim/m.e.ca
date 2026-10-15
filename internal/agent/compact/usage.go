// Purpose: Define cumulative request usage limits and byte-based fallback estimates.
// Pattern: Simple Module.
// Usage: Configuration and task runtime use Settings and TokenCount.
// Related: internal/config/settings.go, internal/agent/compact/controller.go.
package compact

import "fmt"

type Settings struct {
	KeepRecentBytes int   `toml:"keep_recent_bytes"`
	ThresholdTokens int64 `toml:"threshold_tokens"`
	MaxSummaryBytes int   `toml:"max_summary_bytes"`
	MaxInputBytes   int   `toml:"max_input_bytes"`
}

func Defaults() Settings {
	return Settings{KeepRecentBytes: 16 * 1024, ThresholdTokens: 200000, MaxSummaryBytes: 16 * 1024, MaxInputBytes: 64 * 1024}
}
func (s Settings) Check() error {
	if s.KeepRecentBytes < 0 || s.KeepRecentBytes > 1024*1024 || s.ThresholdTokens < 1 || s.ThresholdTokens > 1000000000 || s.MaxSummaryBytes < 1024 || s.MaxSummaryBytes > 64*1024 || s.MaxInputBytes < 8192 || s.MaxInputBytes > 256*1024 || s.MaxSummaryBytes > s.MaxInputBytes/2 {
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
	return TokenCount{Amount: (bytes + 2) / 3, Estimated: true, Kind: "request"}
}

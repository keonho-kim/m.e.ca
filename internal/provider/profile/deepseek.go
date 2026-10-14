// Purpose: Declare reviewed deepseek checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads deepseek records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func deepseek() []Profile {
	return []Profile{
		{
			ID:       "deepseek-v4-flash",
			Family:   "DeepSeek",
			Models:   []string{"deepseek-ai/DeepSeek-V4-Flash", "DeepSeek-V4-Flash"},
			Released: "2026-04-24", TotalB: 284,
			TemplateKeys: []string{"thinking", "enable_thinking"}, VLLMOnly: true,
			Source: "https://huggingface.co/deepseek-ai/DeepSeek-V4-Flash",
			Effort: EffortControl{Kind: ReasoningEffort, Values: []string{"high", "max"}},
		},
	}
}

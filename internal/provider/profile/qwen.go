// Purpose: Declare reviewed qwen checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads qwen records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func qwen() []Profile {
	return []Profile{
		{
			ID: "qwen3.8-flash-next", Family: "Qwen",
			Models: []string{
				"Qwen/Qwen3.8-Flash-Next", "Qwen3.8-Flash-Next",
				"Qwen/Qwen3.8-Flash-Next-FP8", "Qwen3.8-Flash-Next-FP8",
				"Qwen3.8-Flash-Next 180B", "Qwen3.8-Flash-Next-180B",
				"Qwen3.8-Flash-Next-180B-FP8", "Qwen3.8-Flash-Next 180B FP8",
				"Qwen-3.8-180B", "Qwen-3.8-180B-FP8",
			},
			Released: "2026-08-26", TotalB: 180,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.8-Flash-Next",
			Effort:       EffortControl{Kind: ReasoningEffort, Values: []string{"low", "medium", "xhigh"}},
		},
		{
			ID:       "qwen3.5-0.8b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.5-0.8B", "Qwen3.5-0.8B"},
			Released: "2026-03-02", TotalB: 0.8,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.5-0.8B",
		},
		{
			ID:       "qwen3.5-2b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.5-2B", "Qwen3.5-2B"},
			Released: "2026-03-02", TotalB: 2,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.5-2B",
		},
		{
			ID:       "qwen3.5-4b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.5-4B", "Qwen3.5-4B"},
			Released: "2026-03-02", TotalB: 4,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.5-4B",
		},
		{
			ID:       "qwen3.5-9b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.5-9B", "Qwen3.5-9B"},
			Released: "2026-03-02", TotalB: 9,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.5-9B",
		},
		{
			ID:       "qwen3.6-35b-a3b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.6-35B-A3B", "Qwen3.6-35B-A3B"},
			Released: "2026-04-16", TotalB: 35,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.6-35B-A3B",
		},
		{
			ID:       "qwen3.6-27b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.6-27B", "Qwen3.6-27B"},
			Released: "2026-04-22", TotalB: 27,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.6-27B",
		},
		{
			ID:       "qwen3.8-27b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.8-27B", "Qwen3.8-27B"},
			Released: "2026-08-14", TotalB: 27,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/Qwen/Qwen3.8-27B",
			Effort:       EffortControl{Kind: ReasoningEffort, Values: []string{"low", "medium", "xhigh"}},
		},
		{
			ID:       "ornith-1.0-9b",
			Family:   "Qwen",
			Models:   []string{"ornith-ai/Ornith-1.0-9B", "Ornith-1.0-9B"},
			Released: "2026-06", TotalB: 9,
			Source: "https://huggingface.co/ornith-ai/Ornith-1.0-9B",
		},
		{
			ID:       "ornith-1.0-35b",
			Family:   "Qwen",
			Models:   []string{"ornith-ai/Ornith-1.0-35B", "Ornith-1.0-35B"},
			Released: "2026-06", TotalB: 35,
			Source: "https://huggingface.co/ornith-ai/Ornith-1.0-35B",
		},
		{
			ID:       "ornith-1.5-9b",
			Family:   "Qwen",
			Models:   []string{"ornith-ai/Ornith-1.5-9B", "Ornith-1.5-9B"},
			Released: "2026-08", TotalB: 9,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/ornith-ai/Ornith-1.5-9B",
		},
		{
			ID:       "ornith-1.5-35b-a3b",
			Family:   "Qwen",
			Models:   []string{"ornith-ai/Ornith-1.5-35B-A3B", "Ornith-1.5-35B-A3B"},
			Released: "2026-08", TotalB: 35,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/ornith-ai/Ornith-1.5-35B-A3B",
		},
		{
			ID:       "qwen3.5-122b-a10b",
			Family:   "Qwen",
			Models:   []string{"Qwen/Qwen3.5-122B-A10B", "Qwen3.5-122B-A10B"},
			Released: "2026-02-24", TotalB: 122,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://github.com/QwenLM/Qwen3.8",
		},
		{
			ID:       "ornith-1.5-397b",
			Family:   "Qwen",
			Models:   []string{"ornith-ai/Ornith-1.5-397B", "Ornith-1.5-397B"},
			Released: "2026-08", TotalB: 397,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://ornith.ai/ornith_1_5.html",
		},
	}
}

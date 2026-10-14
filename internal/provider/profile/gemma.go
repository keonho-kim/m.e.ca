// Purpose: Declare reviewed gemma checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads gemma records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func gemma() []Profile {
	return []Profile{
		{
			ID:       "gemma-4-e2b-it",
			Family:   "Gemma",
			Models:   []string{"google/gemma-4-E2B-it", "gemma-4-E2B-it"},
			Released: "2026-04-02", TotalB: 5.1,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/google/gemma-4-E2B-it",
		},
		{
			ID:       "gemma-4-e4b-it",
			Family:   "Gemma",
			Models:   []string{"google/gemma-4-E4B-it", "gemma-4-E4B-it"},
			Released: "2026-04-02", TotalB: 8,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/google/gemma-4-E4B-it",
		},
		{
			ID:       "gemma-4-26b-a4b-it",
			Family:   "Gemma",
			Models:   []string{"google/gemma-4-26B-A4B-it", "gemma-4-26B-A4B-it"},
			Released: "2026-04-02", TotalB: 25.2,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/google/gemma-4-26B-A4B-it",
		},
		{
			ID:       "gemma-4-31b-it",
			Family:   "Gemma",
			Models:   []string{"google/gemma-4-31B-it", "gemma-4-31B-it"},
			Released: "2026-04-02", TotalB: 30.7,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/google/gemma-4-31B-it",
		},
		{
			ID:       "gemma-4-12b-it",
			Family:   "Gemma",
			Models:   []string{"google/gemma-4-12B-it", "gemma-4-12B-it"},
			Released: "2026-06-03", TotalB: 11.95,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/google/gemma-4-12B-it",
		},
	}
}

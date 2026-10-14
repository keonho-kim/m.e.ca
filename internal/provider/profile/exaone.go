// Purpose: Declare reviewed exaone checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads exaone records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func exaone() []Profile {
	return []Profile{
		{
			ID:       "exaone-4.5-33b",
			Family:   "EXAONE",
			Models:   []string{"LGAI-EXAONE/EXAONE-4.5-33B", "EXAONE-4.5-33B"},
			Released: "2026-04-09", TotalB: 33,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/LGAI-EXAONE/EXAONE-4.5-33B",
		},
	}
}

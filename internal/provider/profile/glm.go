// Purpose: Declare reviewed glm checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads glm records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func glm() []Profile {
	return []Profile{
		{
			ID:       "glm-4.5-air",
			Family:   "GLM",
			Models:   []string{"zai-org/GLM-4.5-Air", "GLM-4.5-Air"},
			Released: "2025-07", TotalB: 106,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/zai-org/GLM-4.5-Air",
		},
		{
			ID:       "glm-4.7-flash",
			Family:   "GLM",
			Models:   []string{"zai-org/GLM-4.7-Flash", "GLM-4.7-Flash"},
			Released: "2026-01", TotalB: 30,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/zai-org/GLM-4.7-Flash",
		},
		{
			ID:     "glm-5.3-flash",
			Family: "GLM",
			Models: []string{"zai-org/GLM-5.3-Flash", "GLM-5.3-Flash"},
			TotalB: 320,
			Source: "https://huggingface.co/zai-org/GLM-5.3-Flash",
		},
	}
}

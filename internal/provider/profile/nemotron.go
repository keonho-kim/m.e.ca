// Purpose: Declare reviewed nemotron checkpoints and their documented control eligibility.
// Pattern: Family-specific catalogue.
// Usage: Profile resolution reads nemotron records; out-of-range models remain excluded.
// Related: internal/provider/profile/profile.go.
package profile

func nemotron() []Profile {
	return []Profile{
		{
			ID:       "nemotron-3-nano-4b",
			Family:   "Nemotron",
			Models:   []string{"nvidia/NVIDIA-Nemotron-3-Nano-4B-BF16", "NVIDIA-Nemotron-3-Nano-4B-BF16"},
			Released: "2026-03-16", TotalB: 3.97,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Nano-4B-BF16",
		},
		{
			ID:       "nemotron-3-super-120b-a12b",
			Family:   "Nemotron",
			Models:   []string{"nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-FP8", "NVIDIA-Nemotron-3-Super-120B-A12B-FP8", "nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-NVFP4", "NVIDIA-Nemotron-3-Super-120B-A12B-NVFP4"},
			Released: "2026-03-11", TotalB: 120,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16",
			Effort:       EffortControl{Kind: LowEffort, Values: []string{"low", "full"}},
		},
		{
			ID:       "nemotron-3-nano-omni-30b-a3b",
			Family:   "Nemotron",
			Models:   []string{"nvidia/Nemotron-3-Nano-Omni-30B-A3B-Reasoning-FP8", "Nemotron-3-Nano-Omni-30B-A3B-Reasoning-FP8"},
			Released: "2026-04-28", TotalB: 31,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/nvidia/Nemotron-3-Nano-Omni-30B-A3B-Reasoning-FP8",
		},
		{
			ID:       "nemotron-3.5-lightning-30b-a3b",
			Family:   "Nemotron",
			Models:   []string{"nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16", "NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16", "nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-NVFP4", "NVIDIA-Nemotron-3.5-Lightning-30B-A3B-NVFP4"},
			Released: "2026-08-11", TotalB: 30,
			TemplateKeys: []string{"enable_thinking"},
			Source:       "https://huggingface.co/nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16",
		},
	}
}

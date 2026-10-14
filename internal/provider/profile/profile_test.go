// Purpose: Verify precise model identity, explicit serving aliases and catalogue eligibility.
// Pattern: Model-profile boundary tests.
// Usage: go test ./internal/provider/profile.
// Related: internal/provider/profile/profile.go, internal/provider/profile/qwen.go, internal/provider/profile/glm.go.
package profile

import (
	"net/url"
	"testing"
	"time"
)

func TestResolveModelIdentityAndAliases(t *testing.T) {
	for _, test := range []struct {
		model, override, want string
		fail                  bool
	}{
		{"Qwen/Qwen3.6-27B", "", "qwen3.6-27b", false},
		{"qwen3.8-27b", "", "qwen3.8-27b", false},
		{"ornith-ai/Ornith-1.5-35B-A3B", "", "ornith-1.5-35b-a3b", false},
		{"production", "", "generic", false},
		{"someone/Qwen3.6-27B", "", "generic", false},
		{"Qwen3.6-27B-Q4_K_M.gguf", "", "generic", false},
		{"production", "ornith-1.5-9b", "ornith-1.5-9b", false},
		{"Qwen/Qwen3.6-27B", "generic", "generic", false},
		{"Qwen/Qwen3.6-27B", "gemma-4-31b-it", "", true},
		{"production", "typo", "", true},
		{"GLM-4.5-Air", "", "generic", false},
		{"GLM-5.3-Flash", "", "generic", false},
		{"production", "glm-5.3-flash", "", true},
		{"Qwen/Qwen3.5-122B-A10B", "", "generic", false},
	} {
		t.Run(test.model+"/"+test.override, func(t *testing.T) {
			got, err := Resolve(test.model, test.override)
			if (err != nil) != test.fail || err == nil && got.ID != test.want {
				t.Fatalf("profile=%s err=%v; want %s failure=%t", got.ID, err, test.want, test.fail)
			}
		})
	}
}

func TestEligibleUsesReleaseAndTotalParameters(t *testing.T) {
	for _, test := range []struct {
		released string
		total    float64
		want     bool
	}{
		{"2026-02", 120, false}, {"2026-03", 300, true}, {"2026-04", 300.1, false}, {"2026-04", 0, false},
	} {
		if got := (Profile{Released: test.released, TotalB: test.total}).eligible(); got != test.want {
			t.Fatalf("release=%s total=%g eligible=%t", test.released, test.total, got)
		}
	}
}

func TestFamilyProfilesDeclareVerifiedControls(t *testing.T) {
	for _, model := range []string{"Qwen/Qwen3.5-9B", "Qwen/Qwen3.8-27B", "ornith-ai/Ornith-1.5-9B", "nvidia/NVIDIA-Nemotron-3-Super-120B-A12B-BF16", "nvidia/NVIDIA-Nemotron-3.5-Lightning-30B-A3B-BF16", "deepseek-ai/DeepSeek-V4-Flash", "LGAI-EXAONE/EXAONE-4.5-33B", "google/gemma-4-31B-it"} {
		p, err := Resolve(model, "")
		if err != nil || p.ID == "generic" || len(p.TemplateKeys) == 0 {
			t.Fatalf("missing verified profile for %s: %+v %v", model, p, err)
		}
	}
}

func TestProfileReviewFactsAreComplete(t *testing.T) {
	for _, p := range catalogue() {
		source, err := url.Parse(p.Source)
		if err != nil || source.Scheme != "https" || source.Host == "" || p.Family == "" {
			t.Fatalf("missing review source for %s", p.ID)
		}
		if p.Released == "" && p.TotalB > 300 {
			continue
		}
		layout := "2006-01-02"
		if len(p.Released) == 7 {
			layout = "2006-01"
		}
		if _, err := time.Parse(layout, p.Released); err != nil {
			t.Fatalf("invalid release for %s: %v", p.ID, err)
		}
	}
}

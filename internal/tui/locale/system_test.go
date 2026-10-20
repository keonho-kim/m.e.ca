// Purpose: Verify terminal language precedence, fallback and display-only translations.
// Pattern: Behavioral locale tests.
// Usage: go test ./internal/tui/locale.
// Related: internal/tui/locale/system.go, internal/tui/locale/locale.go.
package locale

import "testing"

func TestDetect(t *testing.T) {
	for _, test := range []struct {
		name   string
		env    map[string]string
		native string
		want   Language
	}{
		{"Korean Linux", map[string]string{"LANG": "ko_KR.UTF-8"}, "", Korean},
		{"Korean WSL", map[string]string{"LANG": "ko_KR.utf8"}, "", Korean},
		{"messages override", map[string]string{"LANG": "en_US.UTF-8", "LC_MESSAGES": "ko_KR.UTF-8"}, "", Korean},
		{"all override", map[string]string{"LANG": "ko_KR.UTF-8", "LC_MESSAGES": "ko_KR.UTF-8", "LC_ALL": "en_US.UTF-8"}, "", English},
		{"GNU preference", map[string]string{"LANG": "en_US.UTF-8", "LANGUAGE": "fr:ko:en"}, "", Korean},
		{"C is neutral", map[string]string{"LANG": "C.UTF-8", "LANGUAGE": "ko"}, "ko-KR", English},
		{"native macOS", nil, "ko-KR", Korean},
		{"English native", nil, "en-US", English},
		{"unsupported", map[string]string{"LANG": "ja_JP.UTF-8"}, "ko-KR", English},
		{"absent", nil, "", English},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := detect(func(key string) string { return test.env[key] }, func() string { return test.native })
			if got != test.want {
				t.Fatalf("got %v want %v", got, test.want)
			}
		})
	}
}

func TestModelLabelsArePresentationValues(t *testing.T) {
	if Korean.Text("Thinking") != "추론" {
		t.Fatal("Korean model label missing")
	}
	if English.Text("추론") != "Thinking" {
		t.Fatal("English model label missing")
	}
	if Korean.Text("unregistered literal") != "unregistered literal" {
		t.Fatal("unknown label changed")
	}
}

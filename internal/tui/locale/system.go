// Purpose: Resolve terminal language from locale environment with native OS fallback.
// Pattern: Startup environment adapter with pure locale parsing.
// Usage: Session and conversation selection resolve their language once before rendering.
// Related: internal/tui/locale/system_darwin.go, internal/tui/locale/system_other.go.
package locale

import (
	"context"
	"os"
	"strings"
)

func System(ctx context.Context) Language {
	return detect(os.Getenv, func() string { return nativeLanguage(ctx) })
}

func detect(getenv func(string) string, native func() string) Language {
	name := ""
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		if name = strings.TrimSpace(getenv(key)); name != "" {
			break
		}
	}
	base := strings.ToLower(name)
	if i := strings.IndexAny(base, ".@"); i >= 0 {
		base = base[:i]
	}
	if base == "c" || base == "posix" {
		return English
	}
	if preferred := getenv("LANGUAGE"); preferred != "" {
		for _, candidate := range strings.Split(preferred, ":") {
			if lang, ok := parse(candidate); ok {
				return lang
			}
		}
	}
	if name != "" {
		lang, _ := parse(name)
		return lang
	}
	if native != nil {
		if lang, ok := parse(native()); ok {
			return lang
		}
	}
	return English
}

func parse(name string) (Language, bool) {
	base := strings.ToLower(strings.TrimSpace(name))
	if i := strings.IndexAny(base, "_-.@"); i >= 0 {
		base = base[:i]
	}
	switch base {
	case "ko":
		return Korean, true
	case "en":
		return English, true
	}
	return English, false
}

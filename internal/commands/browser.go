// Purpose: Expose supported browser commands through the same canonical parser and catalog as TUI.
// Pattern: Pure command projection.
// Usage: Native commandCatalog/parseInput feed browser completion and action dispatch.
// Related: internal/commands/catalog.go, internal/commands/parse.go.
package commands

import (
	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func BrowserCatalog() []Suggestion {
	var values []Suggestion
	for _, value := range catalog {
		if value.kind != NewSession && value.kind != Exit {
			values = append(values, Suggestion{value.name, value.description})
		}
	}
	return values
}

func BrowserInput(text string) (string, string, error) {
	c, err := Parse(text)
	if err != nil {
		return "", "", err
	}
	if c.Kind == Message {
		return "message", c.Argument, nil
	}
	if c.Kind == Empty {
		return "empty", "", nil
	}
	for _, value := range catalog {
		if value.kind == c.Kind && c.Kind != NewSession && c.Kind != Exit {
			return value.name[1:], c.Argument, nil
		}
	}
	return "", "", fault.New(diag.InvalidArgument, diag.UICommandUnavailable)
}

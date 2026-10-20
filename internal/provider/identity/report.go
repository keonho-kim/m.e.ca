// Purpose: Represent bounded server identity evidence without transport or model-control policy.
// Pattern: Simple Module.
// Usage: Provider adapters collect facts; setup, SQLite and the TUI consume reports.
// Related: internal/provider/discovery/identify.go, internal/provider/setup/setup.go.
package identity

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type Fact struct {
	Source string `json:"source"`
	Field  string `json:"field"`
	Value  string `json:"value"`
}

// Reports describe server declarations, never prove the actual loaded weights.
type Report struct {
	Failure     string `json:"failure,omitempty"`
	Facts       []Fact `json:"facts,omitempty"`
	Unavailable bool   `json:"unavailable,omitempty"`
	Incomplete  bool   `json:"incomplete,omitempty"`
	Routed      bool   `json:"routed,omitempty"`
}

func safe(value string) bool {
	return value != "" && len(value) <= 2048 && utf8.ValidString(value) && !strings.ContainsFunc(value, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) })
}

func valid(f Fact) bool {
	if !safe(f.Source) || len(f.Source) > 128 || !safe(f.Value) {
		return false
	}
	if strings.Contains(f.Value, "://") {
		u, err := url.Parse(f.Value)
		if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return false
		}
	}
	switch f.Field {
	case "original", "family", "architecture", "parameters", "quantization", "publisher", "parent", "tokenizer", "digest", "thinking", "provider", "upstream", "label":
		return true
	}
	return false
}

// Add rejects unsafe evidence rather than retaining raw metadata or truncated identities.
func (r *Report) Add(source, field, value string) {
	if value == "" {
		return
	}
	f := Fact{source, field, value}
	if slices.Contains(r.Facts, f) {
		return
	}
	if !valid(f) || len(r.Facts) >= 64 {
		r.Incomplete = true
		return
	}
	r.Facts = append(r.Facts, f)
}

func (r Report) Originals() []string {
	var values []string
	for _, f := range r.Facts {
		if f.Field == "original" && !slices.Contains(values, f.Value) {
			values = append(values, f.Value)
		}
	}
	return values
}

func (r Report) State() string {
	originals := r.Originals()
	if r.Routed || len(originals) > 1 {
		return "ambiguous"
	}
	if len(originals) == 1 {
		return "reported"
	}
	for _, f := range r.Facts {
		if f.Field == "family" || f.Field == "architecture" {
			return "family"
		}
	}
	return "unknown"
}

func (r Report) Validate() error {
	if r.Failure != "" && !safe(r.Failure) {
		return fmt.Errorf("invalid model identity failure detail")
	}
	if len(r.Facts) > 64 {
		return fmt.Errorf("too many model identity facts")
	}
	for _, f := range r.Facts {
		if !valid(f) {
			return fmt.Errorf("invalid model identity evidence")
		}
	}
	return nil
}

// Redact removes evidence containing credentials before it is displayed or persisted.
func (r *Report) Redact(redact func(string) string) {
	r.Failure = redact(r.Failure)
	facts := r.Facts[:0]
	for _, f := range r.Facts {
		if redact(f.Value) != f.Value {
			r.Incomplete = true
			continue
		}
		facts = append(facts, f)
	}
	r.Facts = facts
}

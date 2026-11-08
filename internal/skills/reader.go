// Purpose: Define bounded skill reads and resolve explicitly selected skill sources.
// Pattern: Configured filesystem reader with a consumer-supplied access boundary.
// Usage: Local tools, user mentions and server previews share Reader.
// Related: internal/skills/catalog.go, internal/skills/files.go, internal/skills/format.go.
package skills

import (
	"context"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

const (
	MaxMetadataBytes     = 64 << 10
	MaxDocumentBytes     = 256 << 10
	MaxResourceBytes     = 64 << 10
	MaxResourceFileBytes = 1 << 30
	MaxPageBytes         = 16 << 10
	MaxEntries           = 4096
)

var (
	ErrInvalid  = faults.Errorf(diagcatalog.SkillsInvalidSkillRequest)
	ErrNotFound = faults.Errorf(diagcatalog.SkillsSkillNotFound)
	ErrLimit    = faults.Errorf(diagcatalog.SkillsSkillReadLimitExceeded)
	ErrManual   = faults.Errorf(diagcatalog.SkillsSkillRequiresAnExplicitUserMention)
)

type Diagnostic struct{ ID, Message string }
type Page struct {
	Items                []instruction.Metadata `json:"items"`
	NextCursor           string                 `json:"next_cursor,omitempty"`
	Diagnostics          []Diagnostic           `json:"diagnostics,omitempty"`
	DiagnosticsTruncated bool                   `json:"diagnostics_truncated,omitempty"`
}

type Reader struct {
	project, home string
	redact        func(string) string
	check         func(string) error
	mu            sync.RWMutex
	paths         map[string]pathEntry
	pathClock     uint64
	pathBytes     int
	rangeRedact   func(string, int, int) (string, error)
	version       func() int
	now           func() time.Time
	catalogGate   chan struct{}
	cached        *catalogSnapshot
	snapshots     map[string]*catalogSnapshot
}

type Options struct {
	Redact      func(string) string
	RedactRange func(string, int, int) (string, error)
	Version     func() int
	Check       func(string) error
	Now         func() time.Time
}

func New(project, home string, options Options) *Reader {
	if options.Now == nil {
		options.Now = time.Now
	}
	if options.Version == nil {
		options.Version = func() int { return 0 }
	}
	return &Reader{project: project, home: home, redact: options.Redact, check: options.Check, rangeRedact: options.RedactRange, version: options.Version, now: options.Now, catalogGate: make(chan struct{}, 1)}
}

type source struct{ name, path string }

func (r *Reader) sources() []source {
	return []source{{"project", filepath.Join(r.project, ".agents", "skills")}, {"meca", filepath.Join(r.home, ".meca", "skills")}, {"shared", filepath.Join(r.home, ".agents", "skills")}}
}

func (r *Reader) resolve(ctx context.Context, selected instruction.Selector) (string, string, error) {
	if err := ctx.Err(); err != nil {
		return "", "", err
	}
	if (selected.ID == "") == (selected.Path == "") {
		return "", "", faults.Errorf(diagcatalog.SkillsChooseIdOrPath, ErrInvalid)
	}
	if len(selected.ID) > 512 || !utf8.ValidString(selected.ID) || !utf8.ValidString(selected.Path) {
		return "", "", ErrInvalid
	}
	if selected.Path != "" {
		path := selected.Path
		if len(path) > 4096 || strings.ContainsRune(path, 0) {
			return "", "", ErrInvalid
		}
		if path == "~" {
			path = r.home
		} else if strings.HasPrefix(path, "~/") {
			path = filepath.Join(r.home, path[2:])
		}
		if !filepath.IsAbs(path) {
			path = filepath.Join(r.project, path)
		}
		if filepath.Base(path) == "SKILL.md" {
			path = filepath.Dir(path)
		}
		return path, "path", nil
	}
	if scope, name, ok := strings.Cut(selected.ID, ":"); ok {
		if scope == "path" {
			path := r.pathFor(selected.ID)
			if path == "" {
				return "", "", ErrNotFound
			}
			return path, scope, nil
		}
		if name == "" || name == "." || name == ".." || filepath.Base(name) != name || strings.ContainsAny(name, "/\\\x00") || len(name) > 255 {
			return "", "", ErrInvalid
		}
		for _, s := range r.sources() {
			if s.name == scope {
				return filepath.Join(s.path, name), scope, nil
			}
		}
		return "", "", ErrInvalid
	}
	if !instruction.ValidName(selected.ID) {
		return "", "", ErrInvalid
	}
	for _, source := range r.sources() {
		path := filepath.Join(source.path, selected.ID)
		if _, _, err := r.document(ctx, path, source.name, true); err == nil {
			return path, source.name, nil
		} else if ctx.Err() != nil {
			return "", "", ctx.Err()
		}
	}
	return "", "", faults.Errorf(diagcatalog.SkillsWrappedCause, ErrNotFound, selected.ID)
}

func (r *Reader) Load(ctx context.Context, selected instruction.Selector, explicit bool) (instruction.Document, error) {
	path, scope, err := r.resolve(ctx, selected)
	if err != nil {
		return instruction.Document{}, err
	}
	doc, actual, err := r.document(ctx, path, scope, false)
	if err != nil {
		return instruction.Document{}, err
	}
	if doc.ManualOnly && !explicit && !instruction.ExplicitRead(ctx, doc.ID) {
		return instruction.Document{}, ErrManual
	}
	if scope == "path" {
		r.rememberPath(doc.ID, actual)
	}
	return doc, nil
}

func (r *Reader) clean(value string) string {
	if r.redact == nil {
		return ""
	}
	return r.redact(value)
}

func (r *Reader) cleanRange(text string, start, end int) (string, error) {
	redact := r.rangeRedact
	if redact != nil {
		return redact(text, start, end)
	}
	whole, prefix, suffix := r.clean(text), r.clean(text[:start]), r.clean(text[end:])
	if !strings.HasPrefix(whole, prefix) || !strings.HasSuffix(whole, suffix) || len(prefix)+len(suffix) > len(whole) {
		return "", faults.Errorf(diagcatalog.SkillsReferenceRangeCrossesAProtectedValueAdjust, ErrInvalid)
	}
	return whole[len(prefix) : len(whole)-len(suffix)], nil
}

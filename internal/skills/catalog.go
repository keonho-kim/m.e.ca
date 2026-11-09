// Purpose: Discover bounded skill metadata in deterministic source-priority order.
// Pattern: Bounded discovery with snapshot-bound pagination.
// Usage: list_skills, !skills and unqualified names call this reader.
// Related: internal/skills/reader.go, internal/skills/files.go.
package skills

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (r *Reader) catalog(ctx context.Context) ([]catalogEntry, []Diagnostic, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, false, err
	}
	var items []catalogEntry
	var diagnostics []Diagnostic
	seen := make(map[string]bool)
	count, bytes := 0, 0
	truncated := false
	for _, source := range r.sources() {
		root, _, err := r.openRoot(ctx, source.path)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil, nil, false, ctx.Err()
			}
			diagnostics = append(diagnostics, Diagnostic{ID: source.name, Message: diagcatalog.Text(diagcatalog.SkillsSkillSourceIsInaccessible)})
			continue
		}
		file, err := root.Open(".")
		if err != nil {
			return nil, nil, false, errors.Join(err, root.Close())
		}
		var names []string
		for {
			entries, readErr := file.ReadDir(128)
			for _, entry := range entries {
				count++
				if count > MaxEntries {
					return nil, nil, false, errors.Join(ErrLimit, file.Close(), root.Close())
				}
				if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
					names = append(names, entry.Name())
				}
			}
			if readErr != nil {
				if !errors.Is(readErr, io.EOF) {
					return nil, nil, false, errors.Join(readErr, file.Close(), root.Close())
				}
				break
			}
			if err := ctx.Err(); err != nil {
				return nil, nil, false, errors.Join(err, file.Close(), root.Close())
			}
		}
		if err := errors.Join(file.Close(), root.Close()); err != nil {
			return nil, nil, false, err
		}
		sort.Strings(names)
		for _, name := range names {
			doc, actual, err := r.document(ctx, filepath.Join(source.path, name), source.name, true)
			if ctx.Err() != nil {
				return nil, nil, false, ctx.Err()
			}
			if err != nil {
				if len(diagnostics) < 16 {
					message := diagcatalog.Text(diagcatalog.SkillsMetadataUnavailable)
					if errors.Is(err, ErrInvalid) || errors.Is(err, ErrLimit) {
						message = err.Error()
					}
					diagnostics = append(diagnostics, Diagnostic{ID: r.clean(source.name + ":" + name), Message: message})
				} else {
					truncated = true
				}
				continue
			}
			if seen[actual] {
				continue
			}
			seen[actual] = true
			entry := newCatalogEntry(doc.Metadata)
			bytes += entry.bytes
			if bytes > 8<<20 {
				return nil, nil, false, ErrLimit
			}
			items = append(items, entry)
		}
	}
	return items, diagnostics, truncated, ctx.Err()
}

func metadataBytes(m instruction.Metadata) int {
	body, _ := json.Marshal(m)
	return len(body) + 64
}

func (r *Reader) List(ctx context.Context, cursor string, limit int) (Page, error) {
	if limit < 1 || limit > 32 || len(cursor) > 64 {
		return Page{}, ErrInvalid
	}
	var snapshot *catalogSnapshot
	var err error
	start := 0
	if cursor == "" {
		snapshot, err = r.snapshot(ctx, true)
	} else {
		id, offset, ok := strings.Cut(cursor, ":")
		if !ok {
			return Page{}, ErrInvalid
		}
		start, err = strconv.Atoi(offset)
		if err != nil || start < 0 || start > MaxEntries {
			return Page{}, ErrInvalid
		}
		snapshot, err = r.pageSnapshot(ctx, id)
	}
	if err != nil {
		return Page{}, err
	}
	items, diagnostics, truncated := snapshot.items, snapshot.diagnostics, snapshot.truncated
	if start > len(items) {
		return Page{}, ErrInvalid
	}

	page := Page{Items: []instruction.Metadata{}}
	bytes := 128
	if start == 0 {
		page.DiagnosticsTruncated = truncated
		for _, d := range diagnostics {
			body, _ := json.Marshal(d)
			if bytes+len(body)+64 > MaxPageBytes/2 {
				page.DiagnosticsTruncated = true
				break
			}
			page.Diagnostics = append(page.Diagnostics, d)
			bytes += len(body) + 64
		}
	}
	end := start
	for end < len(items) && end-start < limit {
		size := items[end].bytes
		if end == start && bytes+size > MaxPageBytes && len(page.Diagnostics) > 0 {
			page.Diagnostics, page.DiagnosticsTruncated, bytes = nil, true, 128
		}
		if end == start && bytes+size > MaxPageBytes {
			return Page{}, ErrLimit
		}
		if bytes+size > MaxPageBytes {
			break
		}
		page.Items = append(page.Items, cloneMetadata(items[end].Metadata))
		bytes += size
		end++
	}
	if end < len(items) {
		page.NextCursor = snapshot.id + ":" + strconv.Itoa(end)
	}
	return page, nil
}

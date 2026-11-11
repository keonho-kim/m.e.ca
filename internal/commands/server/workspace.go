// Purpose: Expose guarded workspace browsing and persisted session reference selections.
// Pattern: Application use cases over existing filesystem and storage adapters.
// Usage: Native directory, preview and selected-file commands call these operations.
// Related: internal/fileview/reader.go, internal/storage/sqlite/files.go, internal/commands/server/target.go.
package server

import (
	"context"
	requestinput "github.com/keonho-kim/meca/internal/commands/input"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
)

func (s *Service) fileSettings() (fileview.Source, policy.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.options.Files == nil {
		return nil, policy.Settings{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	return s.options.Files, policy.Settings{Scope: policy.System, Permission: s.currentConfig().Permission}, nil
}

func (s *Service) Directory(ctx context.Context, path string, offset, limit int, revision string) (fileview.DirectoryPage, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return fileview.DirectoryPage{}, err
	}
	defer done()
	reader, settings, err := s.fileSettings()
	if err != nil {
		return fileview.DirectoryPage{}, err
	}
	return reader.Directory(ctx, settings, path, offset, limit, revision)
}

func (s *Service) Preview(ctx context.Context, path string) (fileview.Preview, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return fileview.Preview{}, err
	}
	defer done()
	reader, settings, err := s.fileSettings()
	if err != nil {
		return fileview.Preview{}, err
	}
	return reader.Preview(ctx, settings, path)
}

func (s *Service) FileSelections(ctx context.Context, id string) ([]string, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return nil, err
	}
	defer done()
	if s.options.References == nil {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	return s.options.References.List(ctx, id)
}

func (s *Service) SelectFile(ctx context.Context, id, path string, selected bool) error {
	if path == "" || len(path) > 4096 || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return err
	}
	defer done()
	if s.options.References == nil {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	if !selected {
		return s.options.References.Select(ctx, id, path, false)
	}
	files, settings, err := s.fileSettings()
	if err != nil {
		return err
	}
	ctx, release, err := s.beginFileEffect(ctx, settings, path)
	if err != nil {
		return err
	}
	defer release()
	reference, err := files.Reference(ctx, settings, path)
	if err != nil {
		return err
	}
	return s.options.References.Select(ctx, id, reference.Path, true)
}

func (s *Service) resolvePaths(ctx context.Context, text string, paths []string) (string, error) {
	if len(paths) == 0 {
		return text, nil
	}
	if len(paths) > 256 {
		return "", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	reader, settings, err := s.fileSettings()
	if err != nil {
		return "", err
	}
	normalized := make([]string, 0, len(paths))
	for _, path := range paths {
		if len(path) > 4096 || !utf8.ValidString(path) {
			return "", faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
		}
		reference, err := reader.Reference(ctx, settings, path)
		if err != nil {
			return "", err
		}
		normalized = append(normalized, reference.Path)
	}
	return requestinput.Capture(text, normalized).Body(), nil
}

func (s *Service) SteerPaths(ctx context.Context, task, text string, paths []string) error {
	resolved, err := s.resolvePaths(ctx, text, paths)
	if err != nil {
		return err
	}
	return s.Steer(task, resolved)
}

func (s *Service) FileReference(ctx context.Context, path string) (string, error) {
	reader, settings, err := s.fileSettings()
	if err != nil {
		return "", err
	}
	value, err := reader.Reference(ctx, settings, path)
	return value.Path, err
}

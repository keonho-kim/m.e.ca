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

type FileStore interface {
	FileSelections(context.Context, string) ([]string, error)
	SelectFile(context.Context, string, string, bool) error
}

func (s *Service) fileSettings() (fileview.Source, policy.Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.options.Files == nil {
		return nil, policy.Settings{}, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	return s.options.Files, policy.Settings{Scope: policy.System, Permission: s.options.Config.Permission}, nil
}

func (s *Service) Directory(ctx context.Context, path string, offset, limit int) ([]fileview.Entry, bool, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return nil, false, err
	}
	defer done()
	reader, settings, err := s.fileSettings()
	if err != nil {
		return nil, false, err
	}
	return reader.Directory(ctx, settings, path, offset, limit)
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
	c, ctx, done, err := s.beginSessionRead(ctx, id)
	if err != nil {
		return nil, err
	}
	defer done()
	store, ok := c.History.(FileStore)
	if !ok {
		return nil, faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	return store.FileSelections(ctx, id)
}

func (s *Service) SelectFile(ctx context.Context, id, path string, selected bool) error {
	if path == "" || len(path) > 4096 || !utf8.ValidString(path) || strings.ContainsRune(path, 0) {
		return faults.New(diagcatalog.InvalidArgument, diagcatalog.ServerPageInvalid)
	}
	change, err := s.beginTargetChange(ctx, id)
	if err != nil {
		return err
	}
	defer change.finish()
	store, ok := change.conversation.History.(FileStore)
	if !ok {
		return faults.New(diagcatalog.FailedPrecondition, diagcatalog.ServerWorkspaceUnavailable)
	}
	if !selected {
		return store.SelectFile(change.ctx, id, path, false)
	}
	reader, settings, err := s.fileSettings()
	if err != nil {
		return err
	}
	reference, err := reader.Reference(change.ctx, settings, path)
	if err != nil {
		return err
	}
	return store.SelectFile(change.ctx, id, reference.Path, selected)
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

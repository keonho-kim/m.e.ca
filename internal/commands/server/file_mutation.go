// Purpose: Serialize interactive entry mutations with workspace and agent file effects.
// Pattern: Application boundary over a scoped filesystem callback.
// Usage: The native mutateFile command targets one registered workspace.
// Related: internal/fileview/entry_mutation.go, internal/tools/workspacegate/gate.go.
package server

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
	"github.com/keonho-kim/meca/internal/policy"
)

type FileMutationResult struct {
	Key, State      string
	Source          string
	Path            string
	Paths           []string `json:"-"`
	ReferencesReady bool
	Failure         *fault.Failure
}

func (s *Service) mutateEntry(ctx context.Context, session, key string, change fileview.EntryMutation) (FileMutationResult, error) {
	if s.options.MutateEntry == nil || s.options.References == nil || s.options.Files == nil {
		return FileMutationResult{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	if session != "" {
		if _, err := s.options.References.List(ctx, session); err != nil {
			return FileMutationResult{}, err
		}
	}
	s.mu.Lock()
	settings := policy.Settings{Scope: policy.Directory, Permission: s.currentConfig().Permission}
	s.mu.Unlock()
	paths := []string{filepath.Dir(change.Path)}
	if change.Folder != "" {
		paths = append(paths, change.Folder)
	}
	ctx, release, err := s.beginFileEffect(ctx, settings, paths...)
	if err != nil {
		return FileMutationResult{}, err
	}
	defer release()
	source, err := s.options.Files.Reference(ctx, settings, change.Path)
	if err != nil {
		return FileMutationResult{}, err
	}
	path, mutationErr := s.options.MutateEntry(ctx, settings, change)
	if mutationErr != nil && path == "" && !change.Remove {
		return FileMutationResult{}, mutationErr
	}
	result := FileMutationResult{Key: key, Path: path, Source: strings.TrimSuffix(source.Path, "/"), Failure: fileMutationFailure(mutationErr)}
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
	defer cancel()
	err = s.reconcileFileReferences(cleanup, session, settings, &result)
	result.Failure = fileMutationFailure(errors.Join(mutationErr, err))
	return result, nil
}

func (s *Service) reconcileFileReferences(ctx context.Context, session string, settings policy.Settings, result *FileMutationResult) error {
	destination := ""
	if result.Path != "" {
		reference, err := s.options.Files.Reference(ctx, settings, result.Path)
		if err != nil {
			return err
		}
		destination = strings.TrimSuffix(reference.Path, "/")
	}
	err := s.options.References.Rewrite(ctx, func(ctx context.Context, path string) (string, error) {
		if path != result.Source && !strings.HasPrefix(path, result.Source+"/") {
			return path, nil
		}
		if destination != "" {
			return destination + strings.TrimPrefix(path, result.Source), nil
		}
		if result.Failure == nil {
			return "", nil
		}
		_, err := s.options.Files.Reference(ctx, settings, path)
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return path, err
	}, func(id string, paths []string) {
		if id == session {
			result.Paths = paths
		}
		s.mu.Lock()
		s.publish(Event{MutationKey: result.Key, Session: id, Record: eventKind("file_references_changed")})
		s.mu.Unlock()
	})
	if err != nil {
		return err
	}
	result.ReferencesReady = true
	if session != "" {
		result.Paths, err = s.options.References.List(ctx, session)
	}
	return err
}

// Receipts retain registered public diagnostics only, never raw filesystem causes.
func fileMutationFailure(err error) *fault.Failure {
	if err == nil {
		return nil
	}
	code := fault.Code(err)
	return &fault.Failure{Code: code, Detail: diag.Text(diag.PublicMessage(code))}
}

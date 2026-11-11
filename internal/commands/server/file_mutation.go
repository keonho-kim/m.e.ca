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
	Path            string
	Paths           []string
	ReferencesReady bool
	Failure         error
}

type fileSelectionChanges interface {
	FileStore
	ChangeFileSelections(context.Context, string, map[string]string) ([]string, error)
}

func (s *Service) MutateEntry(ctx context.Context, session string, change fileview.EntryMutation) (FileMutationResult, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return FileMutationResult{}, err
	}
	defer done()
	if s.options.MutateEntry == nil {
		return FileMutationResult{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	var store fileSelectionChanges
	if session != "" {
		reservation, err := s.beginTargetChange(ctx, session)
		if err != nil {
			return FileMutationResult{}, err
		}
		defer reservation.finish()
		var ok bool
		store, ok = reservation.conversation.History.(fileSelectionChanges)
		if !ok || s.options.Files == nil {
			return FileMutationResult{}, fault.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
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
	var before []string
	var source string
	if store != nil {
		reference, err := s.options.Files.Reference(ctx, settings, change.Path)
		if err != nil {
			return FileMutationResult{}, err
		}
		source = strings.TrimSuffix(reference.Path, "/")
		before, err = store.FileSelections(ctx, session)
		if err != nil {
			return FileMutationResult{}, err
		}
	}
	path, mutationErr := s.options.MutateEntry(ctx, settings, change)
	result := FileMutationResult{Path: path, Failure: mutationErr, ReferencesReady: store == nil}
	if store == nil {
		return result, nil
	}
	// Filesystem effects cannot roll back with SQL. Finish reference reconciliation
	// even when the browser disconnects, and report incomplete effects explicitly.
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	destination := ""
	if path != "" {
		reference, err := s.options.Files.Reference(cleanup, settings, path)
		if err != nil {
			result.Failure = errors.Join(mutationErr, err)
			return result, nil
		}
		destination = strings.TrimSuffix(reference.Path, "/")
	}
	changes := make(map[string]string)
	for _, selected := range before {
		if selected != source && !strings.HasPrefix(selected, source+"/") {
			continue
		}
		if mutationErr == nil || destination != "" {
			changes[selected] = ""
			if destination != "" {
				changes[selected] = destination + strings.TrimPrefix(selected, source)
			}
		} else if _, err := s.options.Files.Reference(cleanup, settings, selected); errors.Is(err, os.ErrNotExist) {
			// A canceled recursive removal may have removed only some children.
			changes[selected] = ""
		} else if err != nil {
			result.Failure = errors.Join(mutationErr, err)
			return result, nil
		}
	}
	result.Paths, err = store.ChangeFileSelections(cleanup, session, changes)
	result.ReferencesReady = err == nil
	result.Failure = errors.Join(mutationErr, err)
	return result, nil
}

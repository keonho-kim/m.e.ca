// Purpose: Admit file effects once and expose receipts after ambiguous transport failures.
// Pattern: Durable intent with explicit incomplete-result lookup.
// Usage: Native mutateFile and lookupMutation share this service boundary.
// Related: internal/commands/server/file_mutation.go, internal/storage/filemutations.
package server

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/fileview"
)

type FileMutationLedger interface {
	Admit(context.Context, string, string, [32]byte) (FileMutationResult, bool, error)
	Lookup(context.Context, string, string) (FileMutationResult, error)
	Finish(context.Context, string, string, FileMutationResult) error
}

func validMutationIdentity(session, key string) bool {
	return len(session) <= 128 && utf8.ValidString(session) && !strings.ContainsRune(session, 0) && len(key) >= 8 && len(key) <= 128 && utf8.ValidString(key) && !strings.ContainsRune(key, 0)
}

func (s *Service) MutateEntry(ctx context.Context, session, key string, change fileview.EntryMutation) (FileMutationResult, error) {
	if !validMutationIdentity(session, key) {
		return FileMutationResult{}, faults.New(diag.InvalidArgument, diag.FileMutationKey)
	}
	if s.options.Mutations == nil {
		return FileMutationResult{}, faults.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	body, err := json.Marshal(struct {
		Session string
		Change  fileview.EntryMutation
	}{session, change})
	if err != nil {
		return FileMutationResult{}, err
	}
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return FileMutationResult{}, err
	}
	defer done()
	identity := session + "/" + key
	release, err := s.beginMutation(ctx, identity)
	if err != nil {
		return FileMutationResult{}, err
	}
	defer release()
	_, duplicate, err := s.options.Mutations.Admit(ctx, session, key, sha256.Sum256(body))
	if err != nil {
		return FileMutationResult{}, err
	}
	if duplicate {
		release()
		return s.LookupMutation(ctx, session, key)
	}
	result, operationErr := s.mutateEntry(ctx, session, key, change)
	result.Key = key
	if operationErr != nil {
		result.State = "rejected"
		result.Failure = fileMutationFailure(operationErr)
	} else if result.Failure != nil {
		result.State = "partial"
	} else {
		result.State = "applied"
	}
	flush, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := s.options.Mutations.Finish(flush, session, key, result); err != nil {
		return FileMutationResult{}, errors.Join(err, operationErr)
	}
	return result, nil
}

func (s *Service) LookupMutation(ctx context.Context, session, key string) (FileMutationResult, error) {
	if !validMutationIdentity(session, key) {
		return FileMutationResult{}, faults.New(diag.InvalidArgument, diag.FileMutationKey)
	}
	ctx, done, operationErr := s.beginOperation(ctx)
	if operationErr != nil {
		return FileMutationResult{}, operationErr
	}
	defer done()
	if s.options.Mutations == nil {
		return FileMutationResult{}, faults.New(diag.FailedPrecondition, diag.ServerWorkspaceUnavailable)
	}
	value, err := s.options.Mutations.Lookup(ctx, session, key)
	if err != nil {
		return value, err
	}
	if value.State == "pending" {
		s.mu.Lock()
		active := s.fileMutations[session+"/"+key] != nil
		s.mu.Unlock()
		if !active {
			// Completion can commit between the first read and owner cleanup.
			value, err = s.options.Mutations.Lookup(ctx, session, key)
			if err != nil {
				return value, err
			}
			if value.State == "pending" {
				value.State = "uncertain"
			}
		}
	}
	if value.ReferencesReady && session != "" && s.options.References != nil {
		value.Paths, err = s.options.References.List(ctx, session)
	}
	return value, err
}

// A key owns admission as well as filesystem work. Lookups can therefore
// distinguish an intent being committed from a receipt left by an exited owner.
func (s *Service) beginMutation(ctx context.Context, identity string) (func(), error) {
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		s.mu.Lock()
		if s.fileMutations == nil {
			s.fileMutations = make(map[string]chan struct{})
		}
		wait := s.fileMutations[identity]
		if wait == nil {
			owned := make(chan struct{})
			s.fileMutations[identity] = owned
			s.mu.Unlock()
			return sync.OnceFunc(func() { s.mu.Lock(); delete(s.fileMutations, identity); close(owned); s.mu.Unlock() }), nil
		}
		s.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-wait:
		}
	}
}

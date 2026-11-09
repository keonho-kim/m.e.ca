// Purpose: Own cancellable read-only skill queries for the server's fixed project.
// Pattern: Application read use cases.
// Usage: All native transports call these operations through API.Call.
// Related: internal/skills/reader.go, internal/serverapi/skills.go, internal/commands/server/operations.go.
package server

import (
	"context"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (s *Service) ListSkills(ctx context.Context, cursor string, limit int) (skills.Page, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return skills.Page{}, err
	}
	defer done()
	if s.options.Skills == nil {
		return skills.Page{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerSkillReaderIsUnavailable)
	}
	return s.options.Skills.List(ctx, cursor, limit)
}

func (s *Service) GetSkill(ctx context.Context, id string) (instruction.Document, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return instruction.Document{}, err
	}
	defer done()
	if s.options.Skills == nil {
		return instruction.Document{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerSkillReaderIsUnavailable)
	}
	return s.options.Skills.Load(ctx, instruction.Selector{ID: id}, true)
}

func (s *Service) ReadSkillFile(ctx context.Context, id, path string, offset int64, limit int, revision string) (skills.Resource, error) {
	ctx, done, err := s.beginOperation(ctx)
	if err != nil {
		return skills.Resource{}, err
	}
	defer done()
	if s.options.Skills == nil {
		return skills.Resource{}, faults.New(diagcatalog.Unavailable, diagcatalog.ServerSkillReaderIsUnavailable)
	}
	return s.options.Skills.ReadFile(instruction.WithExplicitReads(ctx, []string{id}), id, path, offset, limit, revision)
}

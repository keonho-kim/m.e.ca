// Purpose: Execute skill discovery and reads through the ordinary read-only tool boundary.
// Pattern: Local tool adapter.
// Usage: Prepared skill operations and user/server previews share Skills.
// Related: internal/skills/reader.go, internal/tools/local/prepare.go, internal/tools/local/filesystem/skills.go.
package local

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func (e *Executor) Skills() *skills.Reader { return e.skills }

type skillOperation struct{ operation }

func (op *skillOperation) Execute(ctx context.Context, approved bool) (result toolapi.Result, resultErr error) {
	if err := op.claim(ctx, approved); err != nil {
		return toolapi.Result{}, err
	}
	var value any
	var err error
	input := op.input
	if op.settings.Scope == policy.Directory && (op.spec.Name == "load_skill" && input.Path != "" || strings.HasPrefix(input.SkillID, "path:")) {
		ctx = skills.WithReadRoot(ctx, op.executor.projectRoot)
	}
	switch op.spec.Name {
	case "list_skills":
		limit := 32
		if input.Limit != nil {
			limit = *input.Limit
		}
		value, err = op.executor.skills.List(ctx, input.Cursor, limit)
	case "load_skill":
		var doc instruction.Document
		doc, err = op.executor.skills.Load(ctx, instruction.Selector{ID: input.SkillID, Path: input.Path}, false)
		value = doc
		if err == nil {
			encoded, encodeErr := json.Marshal(doc)
			return toolapi.Result{Content: encoded, Summary: "Read skill " + doc.Name, Skill: &doc}, encodeErr
		}
	case "read_skill_file":
		limit, offset := skills.MaxResourceBytes, 0
		if input.Limit != nil {
			limit = *input.Limit
		}
		if input.Offset != nil {
			offset = *input.Offset
		}
		file, openErr := op.executor.skills.OpenFile(ctx, input.SkillID, input.Path)
		if openErr != nil {
			return toolapi.Result{}, openErr
		}
		defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
		resource, readErr := file.ReadPage(ctx, int64(offset), limit, input.Revision)
		value, err = resource, readErr
		if err == nil && !resource.Binary {
			encoded, encodeErr := json.Marshal(resource)
			return toolapi.Result{Content: encoded, Summary: op.spec.Label, SkillFile: &instruction.ReferenceRead{ID: file.ID, Path: input.Path, Revision: resource.Revision, Offset: resource.Offset}}, encodeErr
		}
		if err == nil && resource.Binary && strings.HasPrefix(resource.MediaType, "image/") && offset == 0 && op.executor.skillImageInput.Load() {
			return op.readImage(ctx, file)
		}
	}
	if err != nil {
		return toolapi.Result{}, err
	}
	encoded, err := json.Marshal(value)
	return toolapi.Result{Content: encoded, Summary: op.spec.Label}, err
}

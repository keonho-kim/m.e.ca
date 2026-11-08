// Purpose: Convert authenticated skill query results into the shared public wire contract.
// Pattern: External read adapter.
// Usage: API.Call dispatches the three read-only skill commands here.
// Related: api/meca/v1/skills.proto, internal/commands/server/skills.go.
package serverapi

import (
	"context"

	v1 "github.com/keonho-kim/meca/internal/serverapi/gen/mecav1"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func wireSkill(m instruction.Metadata) *v1.Skill {
	return &v1.Skill{Id: m.ID, Name: m.Name, Description: m.Description, Source: m.Source, ManualOnly: m.ManualOnly, Warnings: m.Warnings}
}

func (a *API) listSkills(ctx context.Context, input *v1.SkillListRequest, reply *v1.Reply) error {
	limit := int(input.GetLimit())
	if limit == 0 {
		limit = 32
	}
	page, err := a.Service.ListSkills(ctx, input.GetCursor(), limit)
	if err != nil {
		return err
	}
	value := &v1.Skills{NextCursor: page.NextCursor, DiagnosticsTruncated: page.DiagnosticsTruncated}
	for _, m := range page.Items {
		value.Items = append(value.Items, wireSkill(m))
	}
	for _, d := range page.Diagnostics {
		value.Diagnostics = append(value.Diagnostics, &v1.SkillDiagnostic{Id: d.ID, Message: d.Message})
	}
	reply.Result = &v1.Reply_Skills{Skills: value}
	return nil
}

func (a *API) getSkill(ctx context.Context, input *v1.SkillRequest, reply *v1.Reply) error {
	doc, err := a.Service.GetSkill(ctx, input.GetId())
	if err != nil {
		return err
	}
	reply.Result = &v1.Reply_Skill{Skill: &v1.SkillDocument{Skill: wireSkill(doc.Metadata), Revision: doc.Revision, Body: doc.Body}}
	return nil
}

func (a *API) readSkillFile(ctx context.Context, input *v1.SkillFileRequest, reply *v1.Reply) error {
	limit := int(input.GetLimit())
	if limit == 0 {
		limit = skills.MaxResourceBytes
	}
	file, err := a.Service.ReadSkillFile(ctx, input.GetId(), input.GetPath(), input.GetOffset(), limit, input.GetRevision())
	if err != nil {
		return err
	}
	reply.Result = &v1.Reply_SkillFile{SkillFile: &v1.SkillFile{Text: file.Text, MediaType: file.MediaType, Offset: file.Offset, NextOffset: file.NextOffset, TotalBytes: file.TotalBytes, Binary: file.Binary, Revision: file.Revision}}
	return nil
}

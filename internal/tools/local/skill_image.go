// Purpose: Decode skill image assets using the existing vision limits and owned decoder slot.
// Pattern: Read-only image adapter.
// Usage: read_skill_file delegates binary images here for vision-capable requests.
// Related: internal/tools/local/imagefile/decode.go, internal/skills/assets.go.
package local

import (
	"context"
	"encoding/json"

	"github.com/keonho-kim/meca/internal/agent/media"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/skills"
	"github.com/keonho-kim/meca/internal/tools/local/imagefile"
)

func (e *Executor) SetSkillImageInput(enabled bool) { e.skillImageInput.Store(enabled) }

func (op *skillOperation) readImage(ctx context.Context, file *skills.ResourceFile) (toolapi.Result, error) {
	select {
	case op.executor.imageReads <- struct{}{}:
		defer func() { <-op.executor.imageReads }()
	case <-ctx.Done():
		return toolapi.Result{}, ctx.Err()
	}
	body, err := file.ReadBytes(ctx, media.MaxBytes)
	if err != nil {
		return toolapi.Result{}, err
	}
	image, err := imagefile.Decode(ctx, "skill:"+op.input.SkillID+"/"+op.input.Path, body)
	if err != nil {
		return toolapi.Result{}, err
	}
	metadata, err := json.Marshal(struct {
		MIME          string `json:"media_type"`
		Width, Height int
	}{image.MIME, image.Width, image.Height})
	return toolapi.Result{Images: []media.Image{image}, Content: metadata, Summary: "Skill image read"}, err
}

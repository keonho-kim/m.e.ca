// Purpose: Keep cached code tool declarations consistent with provider capability recovery.
// Pattern: Scripted provider rejection followed by an observed model request.
// Usage: go test ./internal/agent/run -run TestCodeCatalogRefreshesAfterImageRejection.
// Related: internal/agent/run/images.go, internal/agent/run/context.go.
package run

import (
	"context"
	"strings"
	"testing"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	"github.com/keonho-kim/meca/internal/policy"
)

type imageCatalogTools struct{ *executor }

func (e imageCatalogTools) Definitions() []toolapi.Definition {
	return append(e.executor.Definitions(), toolapi.Definition{Name: "view_image", Description: "Read image pixels", Effect: policy.Read})
}

type imageRejectingModel struct{ requests []modelapi.Request }

func (g *imageRejectingModel) Generate(_ context.Context, request modelapi.Request, emit func(modelapi.Event) error) error {
	g.requests = append(g.requests, request)
	if len(g.requests) == 1 {
		return &modelapi.ImageInputError{}
	}
	return emit(modelapi.Event{Delta: "Image input is unavailable", Done: true})
}

func TestCodeCatalogRefreshesAfterImageRejection(t *testing.T) {
	g := &imageRejectingModel{}
	r := fixture(&generator{}, &executor{}, &journal{})
	r.Generator, r.Tools, r.ImageInput = g, imageCatalogTools{&executor{}}, true
	if _, err := r.Run(t.Context(), "Inspect available input", approve, emit); err != nil {
		t.Fatal(err)
	}
	if len(g.requests) != 2 {
		t.Fatalf("requests=%d", len(g.requests))
	}
	for i, request := range g.requests {
		if len(request.Tools) != 1 {
			t.Fatal("missing code tool declaration")
		}
		available := strings.Contains(request.Tools[0].Description, "view_image(args:")
		if available != (i == 0) {
			t.Fatalf("request %d retained stale image capability", i)
		}
	}
}

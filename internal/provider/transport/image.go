// Purpose: Recognize explicit image-input rejection without confusing invalid files or transient failures.
// Pattern: Structured provider error classification.
// Usage: Connection probes and generation translate only definitive unsupported-input errors.
// Related: internal/provider/transport/errors.go, internal/provider/diagnostic/image.go.
package transport

import (
	"errors"
	"strings"
)

// ImageUnsupported recognizes explicit model or serving API image-input rejection.
func ImageUnsupported(err error) bool {
	var failure *responseError
	if !errors.As(err, &failure) || failure.status != 0 && failure.status != 400 && failure.status != 404 && failure.status != 422 {
		return false
	}
	switch failure.code {
	case "unsupported_image_input", "image_input_not_supported", "model_does_not_support_images":
		return true
	}
	text := strings.ToLower(failure.message)
	// A format, count or size restriction does not revoke image input as a whole.
	for _, restriction := range []string{"multiple images", "more than one image", "images larger than", "image larger than", "images bigger than", "image size"} {
		if strings.Contains(text, restriction) {
			return false
		}
	}
	for _, phrase := range []string{"does not support image", "doesn't support image", "does not support vision", "image input is not supported", "images are not supported", "image input is unsupported", "image_url is not supported", "does not support multimodal inputs", "multimodal inputs are not supported", "no endpoints found that support image input", "no endpoints found that support base64 image input", "at most 0 image", "image modality is disabled"} {
		if strings.Contains(text, phrase) {
			return true
		}
	}
	return false
}

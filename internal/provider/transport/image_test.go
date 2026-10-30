// Purpose: Separate unsupported image input from image size/count restrictions and other API failures.
// Pattern: Provider failure classification tests.
// Usage: go test ./internal/provider/transport -run TestImage.
// Related: internal/provider/transport/image.go.
package transport

import (
	"encoding/json"
	"testing"
)

func TestImageRejectionClassification(t *testing.T) {
	for _, tc := range []struct {
		message string
		status  int
		want    bool
	}{
		{"The model does not support images", 400, true},
		{"No endpoints found that support image input", 404, true},
		{"At most 0 image(s) may be provided", 400, true},
		{"Multiple images are not supported", 400, false},
		{"The model does not support images larger than 20 MB", 400, false},
		{"Invalid image encoding", 400, false},
		{"Not authorized", 401, false},
		{"Image backend overloaded", 503, false},
	} {
		body, _ := json.Marshal(map[string]any{"error": map[string]string{"message": tc.message}})
		if got := ImageUnsupported(ParseError(tc.status, body, "")); got != tc.want {
			t.Errorf("%q classified %v", tc.message, got)
		}
	}
}

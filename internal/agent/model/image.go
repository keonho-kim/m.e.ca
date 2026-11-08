// Purpose: Define explicit image omission and recoverable input-support rejection.
// Pattern: Model-bound observation values.
// Usage: Context marks absent pixels; providers report rejection to the request owner.
// Related: internal/provider/httpclient/request.go, internal/agent/run/images.go.
package model

import diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"

// ImageOmissionNotice records that retained image pixels are absent from a text-only context.
const ImageOmissionNotice = "\n[Image pixels not provided: image input is unavailable for the selected model.]"

type ImageInputError struct{ Cause error }

func (e *ImageInputError) Error() string {
	if e.Cause != nil {
		return diagcatalog.Text(diagcatalog.ImageInputUnavailableCause, e.Cause.Error())
	}
	return diagcatalog.Text(diagcatalog.ImageInputUnconfirmed)
}
func (e *ImageInputError) Unwrap() error { return e.Cause }

// Purpose: Define explicit image omission and recoverable input-support rejection.
// Pattern: Model-bound observation values.
// Usage: Context marks absent pixels; providers report rejection to the request owner.
// Related: internal/provider/httpclient/request.go, internal/agent/run/images.go.
package model

// ImageOmissionNotice records that retained image pixels are absent from a text-only context.
const ImageOmissionNotice = "\n[Image pixels not provided: image input is unavailable for the selected model.]"

type ImageInputError struct{ Cause error }

func (e *ImageInputError) Error() string {
	if e.Cause != nil {
		return "image input unavailable: " + e.Cause.Error()
	}
	return "image input has not been confirmed for the selected model"
}
func (e *ImageInputError) Unwrap() error { return e.Cause }

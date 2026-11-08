// Purpose: Validate persisted conversation messages and tool result references.
// Pattern: Simple Module.
// Usage: History adapters call Message.Check at decoding and write boundaries.
// Related: internal/agent/model/model.go, internal/agent/tool/call.go.
package model

import (
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

func (message Message) Check() error {
	if message.Source != nil {
		if message.Role != "user" && message.Role != "system" && message.Role != "tool" {
			return faults.Errorf(diagcatalog.ModelPublicProvenanceRequiresAnObservationOrUser)
		}
		if err := message.Source.Check(); err != nil {
			return err
		}
		if message.Role != "tool" && (message.Role == "user") != (message.Source.Author == "user") {
			return faults.Errorf(diagcatalog.ModelMessageRoleAndPublicProvenanceDisagree)
		}
	}
	if message.Role != "user" && message.Role != "assistant" && message.Role != "tool" && message.Role != "system" {
		return faults.Errorf(diagcatalog.ModelInvalidMessageRole)
	}
	if message.Incomplete && (message.Role != "assistant" || len(message.Calls) != 0 || message.ToolCallID != "" || message.ToolName != "") {
		return faults.Errorf(diagcatalog.ModelUnfinishedDraftsMustBeAssistantMessagesWithout)
	}
	if len(message.ReasoningState) > 0 && (message.Role != "assistant" || message.Incomplete) {
		return faults.Errorf(diagcatalog.ModelContinuationStateRequiresACompletedAssistantMessage)
	}
	if len(message.Images) > 0 && message.Role != "user" && message.Role != "tool" {
		return faults.Errorf(diagcatalog.ModelImagesRequireUserOrToolMessages)
	}
	for _, image := range message.Images {
		if err := image.Check(); err != nil {
			return err
		}
	}
	size := message.ContextBytes()
	if size > MaxResponseBytes || len(message.Calls) > toolapi.MaxCalls {
		return faults.Errorf(diagcatalog.ModelMessageExceedsSizeLimits)
	}
	for _, call := range message.Calls {
		if err := call.Check(); err != nil {
			return err
		}
	}
	if message.Role == "tool" && (message.ToolCallID == "" || len(message.ToolCallID) > 128) {
		return faults.Errorf(diagcatalog.ModelInvalidToolResultID)
	}
	return nil
}

// ContextBytes measures retained model content without serializing native state
// or base64 images. Image envelopes reserve the same allowance as model context.
func (message Message) ContextBytes() int {
	size := len(message.Content) + len(message.Reasoning) + len(message.ReasoningState) + len(message.ToolCallID) + len(message.ToolName)
	for _, image := range message.Images {
		size += len(image.Data)*4/3 + 4096
	}
	for _, call := range message.Calls {
		size += len(call.ID) + len(call.Name) + len(call.Arguments)
	}
	return size
}

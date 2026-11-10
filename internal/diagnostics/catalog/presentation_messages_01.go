// Purpose: Own public error messages and actionable failure details.
// Pattern: Declarative diagnostic catalog.
// Usage: Transport and tool callers render these registered templates.
// Related: internal/diagnostics/catalog/messages.go, internal/diagnostics/fault/error.go.
package catalog

const (
	ServerapiAuthenticateOnceThenSendCommandOr Message = "authenticate once, then send command or watch"
	ServerapiAuthenticationRequired            Message = "authentication required"
	ServerapiBearerTokenIsRequired             Message = "bearer token is required"
	ServerapiCommandIDIsRequired               Message = "command ID is required"
	ServerapiCouldNotEncodeReply               Message = "could not encode reply"

	ServerapiExpectedProtobufJSONFrame Message = "expected Protobuf JSON frame"
	ServerapiHeaderIsNotAllowed        Message = "header is not allowed"
	ServerapiHostIsNotAllowed          Message = "Host is not allowed"
	ServerapiMethodIsNotAllowed        Message = "method is not allowed"

	ServerapiOriginIsNotAllowed                Message = "Origin is not allowed"
	ServerapiOriginIsRequired                  Message = "Origin is required"
	ServerapiQueryParametersAreNotSupported    Message = "query parameters are not supported"
	ServerapiServerInputMemoryBudgetIsOccupied Message = "server input memory budget is occupied"
	ServerapiTooManyConnections                Message = "too many connections"
	ServerapiTooManyObservers                  Message = "too many observers"
	ServerapiTooManySockets                    Message = "too many sockets"
	ServerapiWatchRequestExceedsBytes          Message = "watch request exceeds 4096 bytes"
	TeamNothingExecutedS                       Message = "Nothing executed. %s"
	TeamYourCommandSIsStillRunning             Message = "Your command %s is still running; this file change did not start."
)

// Purpose: Own server startup access messages and output failures.
// Pattern: Central presentation catalog.
// Usage: The executable prints server credentials and the UI address after the listener is ready.
// Related: cmd/meca/server.go, cmd/meca/server_ui.go.
package catalog

const (
	ServerTokenReady          Message = "Server token: %s"
	ServerStartupOutputFailed Message = "cannot write server startup output: %w"
	UIReady                   Message = "Web UI: %s"
)

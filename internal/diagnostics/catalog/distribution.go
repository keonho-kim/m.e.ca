// Purpose: Own distribution requirements and variant-aware command help.
// Pattern: Central diagnostic catalog.
// Usage: Startup and executable presentation share these messages.
// Related: internal/commands/startup/server.go, cmd/meca/help.go, cmd/meca/server_minimum.go.
package catalog

const (
	DistributionFullRequired Message = "server and WebUI require the full distribution; install the full release"
	MecaHelpCommon           Message = "M.E.CA (%s)\nUsage: meca [path] [--print|--mode json] [--session ID] [prompt]\n       meca --session ID --export\n       meca --diagnose"
	MecaHelpServer           Message = "       meca [path] --server [--ui] [--protocol sse|socket|grpc] [--listen IP:port]\n            [--session ID] [--concurrency N] [--token-file PATH] [--tls-cert PATH --tls-key PATH]\n            [--allow-host host:port] [--allow-origin http(s)://host:port]"
	MecaHelpMaintenance      Message = "       meca install [--no-shell]\n       meca resume | upgrade | uninstall"
)

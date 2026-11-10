// Purpose: Own distribution requirements, command help and upgrade result messages.
// Pattern: Central diagnostic catalog.
// Usage: Startup and executable presentation share these messages.
// Related: internal/commands/startup/server.go, cmd/meca/help.go, cmd/meca/server_tui.go.
package catalog

const (
	DistributionFullRequired       Message = "server and WebUI require the full distribution; run meca upgrade --full or install the full release"
	MecaHelpCommon                 Message = "M.E.CA (%s)\nUsage: meca [path] [--print|--mode json] [--session ID] [prompt]\n       meca --session ID --export\n       meca --diagnose"
	MecaHelpServer                 Message = "       meca [path] --server [--ui] [--protocol sse|socket|grpc] [--listen IP:port]\n            [--session ID] [--concurrency N] [--token-file PATH] [--tls-cert PATH --tls-key PATH]\n            [--allow-host host:port] [--allow-origin http(s)://host:port]"
	MecaHelpMaintenance            Message = "       meca install [--no-shell]\n       meca upgrade [--tui|--full]\n       meca resume | uninstall"
	UpgradeUnsupportedDistribution Message = "unsupported release distribution: expected tui or full"
	UpgradeCheckingDistribution    Message = "M.E.CA %s 최신 릴리스를 확인합니다…"
	UpgradeDistributionCurrent     Message = "이미 최신 %s 릴리스입니다: %s"
	UpgradeDistributionComplete    Message = "M.E.CA %s 업데이트 완료: %s"
	UpgradePromptsRefreshed        Message = "기본 프롬프트를 갱신했습니다."
)

module github.com/keonho-kim/meca

go 1.27.4

require (
	connectrpc.com/connect/v2 v2.0.2
	github.com/bytedance/sonic v1.15.5
	github.com/coder/websocket v1.8.15
	github.com/creack/pty v1.1.24
	github.com/fsnotify/fsnotify v1.10.1
	github.com/gdamore/tcell/v3 v3.6.0
	github.com/goccy/go-json v0.12.0
	github.com/godbus/dbus/v5 v5.2.4
	github.com/google/go-cmp v0.7.0
	github.com/klauspost/compress v1.20.4
	github.com/pelletier/go-toml/v2 v2.4.3
	github.com/rivo/uniseg v0.4.7
	github.com/yuin/goldmark/v2 v2.1.15
	go.uber.org/goleak v1.3.0
	go.yaml.in/yaml/v3 v3.0.6
	golang.design/x/clipboard v0.11.0
	golang.org/x/image v0.48.0
	golang.org/x/sys v0.50.0
	golang.org/x/term v0.48.0
	google.golang.org/protobuf v1.36.12
	modernc.org/sqlite v1.62.0
	mvdan.cc/sh/v3 v3.14.1
)

require (
	github.com/bytedance/gopkg v0.1.4 // indirect
	github.com/bytedance/sonic/loader v0.5.3 // indirect
	github.com/clipperhouse/displaywidth v0.11.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/cloudwego/base64x v0.1.7 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/ebitengine/purego v0.10.1 // indirect
	github.com/gdamore/encoding v1.0.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/cpuid/v2 v2.4.2 // indirect
	github.com/lucasb-eyer/go-colorful v1.4.1 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	golang.design/x/x11 v0.2.0 // indirect
	golang.org/x/arch v0.33.0 // indirect
	golang.org/x/text v0.44.0 // indirect
	// Match the libc ABI required by modernc.org/sqlite.
	modernc.org/libc v1.80.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
)

replace golang.design/x/clipboard => ./third_party/clipboard

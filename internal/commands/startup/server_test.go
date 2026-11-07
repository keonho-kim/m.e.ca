// Purpose: Verify server startup is explicit and cannot consume interactive or one-shot input.
// Pattern: Startup contract tests.
// Usage: go test ./internal/commands/startup.
// Related: internal/commands/startup/parse.go, internal/commands/startup/server.go.
package startup

import "testing"

func TestServerModes(t *testing.T) {
	for _, protocol := range []string{"sse", "socket", "grpc"} {
		mode, err := Parse([]string{"--server", "--protocol", protocol, "--listen", "127.0.0.1:0", "--openai"}, t.TempDir())
		if err != nil || !mode.Server || mode.ServerOptions.Protocol != protocol || mode.Print || !mode.ServerOptions.OpenAI {
			t.Fatalf("%s: %+v %v", protocol, mode, err)
		}
	}
	mode, err := Parse([]string{"--server"}, t.TempDir())
	if err != nil || mode.ServerOptions.Protocol != "sse" || mode.ServerOptions.Listen != "127.0.0.1:8787" {
		t.Fatal(mode, err)
	}
}

func TestServerRejectsAmbiguousModes(t *testing.T) {
	for _, args := range [][]string{
		{"--openai"},
		{"--protocol", "sse"}, {"--listen", "127.0.0.1:0"},
		{"--server", "--print", "hello"}, {"--server", "--mode", "json"},
		{"--server", "--diagnose"}, {"--server", "--export"},
		{"--server", "--protocol", "unknown"}, {"--server", "--tls-cert", "cert.pem"},
		{"--server", "--protocol", "sse|socket"},
	} {
		if _, err := Parse(args, t.TempDir()); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

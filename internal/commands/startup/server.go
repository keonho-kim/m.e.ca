// Purpose: Parse server transport and network options without activating them in other modes.
// Pattern: Startup boundary.
// Usage: Parse registers and checks these flags before resolving the project directory.
// Related: internal/commands/startup/parse.go, internal/serverapi/network.go.
package startup

import (
	"flag"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
)

type ServerOptions struct {
	Protocol, Listen, TokenFile, Certificate, Key string
	Hosts, Origins                                []string
	OpenAI                                        bool
}

func serverFlags(flags *flag.FlagSet, mode *Mode) {
	flags.BoolVar(&mode.Server, "server", false, "serve the selected project without a terminal")
	o := &mode.ServerOptions
	flags.BoolVar(&o.OpenAI, "openai", false, "add text OpenAI-compatible HTTP endpoints to the server")
	flags.StringVar(&o.Protocol, "protocol", "sse", "server transport: sse, socket (WebSocket), grpc")
	flags.StringVar(&o.Listen, "listen", "127.0.0.1:8787", "server TCP address")
	flags.StringVar(&o.TokenFile, "token-file", "", "private bearer-token file; defaults to ~/.meca/server.token")
	flags.StringVar(&o.Certificate, "tls-cert", "", "TLS certificate PEM file")
	flags.StringVar(&o.Key, "tls-key", "", "TLS private-key PEM file")
	flags.Func("allow-host", "additional exact HTTP Host authority; repeatable", func(value string) error {
		o.Hosts = append(o.Hosts, value)
		return nil
	})
	flags.Func("allow-origin", "allowed browser origin (scheme://host:port); repeatable", func(value string) error {
		o.Origins = append(o.Origins, value)
		return nil
	})
}

func checkServerFlags(flags *flag.FlagSet, mode Mode) error {
	serverFlag := false
	flags.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "protocol", "listen", "token-file", "tls-cert", "tls-key", "allow-host", "allow-origin", "openai":
			serverFlag = true
		}
	})
	if !mode.Server && serverFlag {
		return faults.Errorf(diagcatalog.StartupServerOptionsRequireServer)
	}
	if !mode.Server {
		return nil
	}
	if mode.Print || mode.Output != "text" || mode.Export || mode.Diagnose {
		return faults.Errorf(diagcatalog.StartupServerCannotBeCombinedWithOneShot)
	}
	switch mode.ServerOptions.Protocol {
	case "sse", "socket", "grpc":
	default:
		return faults.Errorf(diagcatalog.StartupProtocolMustBeSseSocketOrGrpc)
	}
	if (mode.ServerOptions.Certificate == "") != (mode.ServerOptions.Key == "") {
		return faults.Errorf(diagcatalog.StartupTlsCertAndTlsKeyMustBe)
	}
	return nil
}

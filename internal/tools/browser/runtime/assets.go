// Purpose: Select the pinned helper and construct versioned official Chrome distribution paths.
// Pattern: Platform distribution manifest.
// Usage: Ensure selects the helper; Stable discovery selects a Chrome version for its platform.
// Related: internal/tools/browser/runtime/install.go, docs/browser.md.
package runtime

import (
	"os"
	"path/filepath"
	"runtime"

	diag "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	"github.com/keonho-kim/meca/internal/diagnostics/fault"
)

const AgentVersion = "0.43.0"

type asset struct {
	name, url, digest, executable, version string
	zip                                    bool
}

func agentAsset(goos, arch string, musl bool) (asset, error) {
	target := goos + "-" + arch
	if arch == "amd64" {
		target = goos + "-x64"
	}
	if goos == "linux" && musl {
		target = "linux-musl-" + arch
		if arch == "amd64" {
			target = "linux-musl-x64"
		}
	}
	hashes := map[string]string{
		"darwin-arm64":     "05338a8eeddf0b3f9ebf0c87a6c21aedc10685608c68618e1a8483c43eae9c3f",
		"darwin-x64":       "49a5ab27297213c1bc0005701a6a4542399abfc9d2cb7520ab519ebb94b8b13d",
		"linux-arm64":      "34be55c2b92d8d199fb9d1d62a905ee5b6ac5b1f3e8c24e5b20597f370d905e5",
		"linux-x64":        "8e69c9fb662514c9cf36e4911e94e913742ba066e55102486350e049dd847612",
		"linux-musl-arm64": "f1d146ad95c1df627b3055475fcb971dfd02f757a1eef756a8f2bd9e19ffff4a",
		"linux-musl-x64":   "10b3b1d5fa8f53f7bc44273bcefaa64299ab77071bc83e75d96de5ed66025c89",
	}
	hash := hashes[target]
	if hash == "" {
		return asset{}, fault.Errorf(diag.BrowserUnsupported, target)
	}
	return asset{name: "agent-browser-" + AgentVersion + "-" + target, url: "https://github.com/vercel-labs/agent-browser/releases/download/v" + AgentVersion + "/agent-browser-" + target, digest: hash, executable: "agent-browser"}, nil
}

func chromeAsset(goos, arch, version string) (asset, error) {
	platform := map[string]string{"darwin/arm64": "mac-arm64", "darwin/amd64": "mac-x64", "linux/amd64": "linux64", "linux/arm64": "linux-arm64"}[goos+"/"+arch]
	if platform == "" {
		return asset{}, fault.Errorf(diag.BrowserUnsupported, goos+"/"+arch)
	}
	if !validVersion(version) {
		return asset{}, fault.Errorf(diag.BrowserStableMetadata)
	}
	executable := "chrome-" + platform + "/chrome"
	if goos == "darwin" {
		executable = "chrome-" + platform + "/Google Chrome for Testing.app/Contents/MacOS/Google Chrome for Testing"
	}
	return asset{name: "chrome-" + version + "-" + platform, version: version, url: "https://storage.googleapis.com/chrome-for-testing-public/" + version + "/" + platform + "/chrome-" + platform + ".zip", executable: executable, zip: true}, nil
}

func hostAgent() (asset, error) {
	musl, _ := filepath.Glob("/lib/ld-musl-*.so.1")
	return agentAsset(runtime.GOOS, runtime.GOARCH, len(musl) > 0)
}
func executable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0111 != 0
}

// Purpose: Verify transfer argument boundaries, concealed payload options and download preservation.
// Pattern: Table-driven protocol parser tests.
// Usage: go test ./internal/policy/command/transfer.
// Related: internal/policy/command/transfer/curl.go, internal/policy/command/transfer/wget.go, internal/policy/command/transfer/remote.go.
package transfer

import (
	"slices"
	"testing"
)

func arguments(values ...string) []Argument {
	result := make([]Argument, len(values))
	for i, value := range values {
		result[i] = Argument{Value: value, Known: true}
	}
	return result
}

func TestHTTPTransfers(t *testing.T) {
	const target = "https://upload.example.test/api"
	for _, tc := range []struct {
		name               string
		args               []Argument
		transfer, indirect bool
	}{
		{"curl", arguments("-fsSL", target, "-o", "archive"), false, false},
		{"curl", arguments("-sS", "--connect-timeout", "5", "--retry", "3", target), false, false},
		{"curl", arguments("--url="+target, "-XPUT", "--data-binary", "fixture"), true, false},
		{"curl", arguments("-d@source", target), true, false},
		{"curl", arguments("-GsSFfile=@source", target), true, false},
		{"curl", arguments("--json={\"fixture\":true}", target), true, false},
		{"curl", arguments("-X", "GET", "--data-urlencode", "file@source", target), true, false},
		{"curl", arguments("--url-query", "file@source", target), true, false},
		{"curl", arguments("-H", "X-Arbitrary: fixture", target), true, false},
		{"curl", arguments(target + "?fixture=value"), true, false},
		{"curl", arguments("--config", "settings", target), false, true},
		{"curl", arguments("--expand-data", "{{secret}}", target), false, true},
		{"curl", arguments("--dat", "fixture", target), false, true},
		{"curl", arguments("--netrc", target), false, true},
		{"curl", arguments("--upload-file", "source", "--location", target), true, true},
		{"curl", arguments("-d", "fixture", "--connect-to", "host:443:elsewhere:443", target), true, true},
		{"curl", arguments("-d", "fixture", "--proxy", "https://proxy.example.test", target), true, true},
		{"wget", arguments("-qO", "archive", target), false, false},
		{"wget", arguments("--post-file=source", target), true, true},
		{"wget", arguments("--body-data=fixture", "--method=GET", "--max-redirect=0", target), true, false},
		{"wget", arguments("--config", "settings", target), false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			result := Inspect(tc.name, tc.args)
			if (len(result.Destinations) > 0) != tc.transfer || result.Indirect != tc.indirect {
				t.Fatalf("%+v: %+v", tc.args, result)
			}
			if tc.transfer && !slices.Contains(result.Destinations, target) && !slices.Contains(result.Destinations, target+"?fixture=value") {
				t.Fatalf("lost actual target: %+v", result)
			}
		})
	}
	result := Inspect("curl", []Argument{{Value: "-T", Known: true}, {}, {Value: target, Known: true}})
	if !slices.Contains(result.Destinations, target) {
		t.Fatal("runtime file name hid explicit upload")
	}
	result = Inspect("curl", []Argument{{}, {Value: target, Known: true}})
	if !result.Indirect {
		t.Fatal("dynamic flags became an automatic download")
	}
}

func TestRemoteTransfers(t *testing.T) {
	for _, tc := range []struct {
		name     string
		args     []Argument
		want     string
		indirect bool
	}{
		{"scp", arguments("-r", "source", "user@host:/srv/app/file"), "ssh://user@host/srv/app/file", false},
		{"scp", arguments("user@host:/srv/file", "destination"), "", false},
		{"scp", arguments("-P", "2222", "source", "user@host:/srv/file"), "ssh://user@host:2222/srv/file", false},
		{"scp", arguments("-P2222", "source", "user@host:file"), "ssh://user@host:2222/~/file", false},
		{"rsync", arguments("-av", "source", "host::module/file"), "rsync://host/module/file", false},
		{"rsync", arguments("-av", "source", "destination"), "", false},
		{"rsync", arguments("-e", "ssh -J proxy", "source", "user@host:/srv/file"), "ssh://user@host/srv/file", true},
		{"ssh", arguments("-p", "2222", "-l", "deploy", "host", "echo", "hello"), "ssh://deploy@host:2222/", false},
		{"ssh", arguments("-o", "ProxyCommand=anything", "user@host", "echo"), "ssh://user@host/", true},
	} {
		result := Inspect(tc.name, tc.args)
		if result.Indirect != tc.indirect || tc.want == "" && len(result.Destinations) != 0 || tc.want != "" && !slices.Equal(result.Destinations, []string{tc.want}) {
			t.Fatalf("%s %+v: %+v", tc.name, tc.args, result)
		}
	}
}

func TestGitPushDestinations(t *testing.T) {
	for _, tc := range []struct {
		args     []Argument
		want     string
		indirect bool
	}{
		{arguments("origin", "main"), "", true},
		{nil, "", true},
		{arguments("./repository", "main"), "", false},
		{arguments("file:///repository", "main"), "", false},
		{arguments("--force-with-lease", "https://host/owner/repo.git", "main"), "https://host/owner/repo.git", false},
		{arguments("--repo=ssh://user@host/repo.git", "main"), "ssh://user@host/repo.git", false},
		{arguments("user@host:/repo.git", "main"), "ssh://user@host/repo.git", false},
		{arguments("ext::program", "main"), "", true},
	} {
		result := GitPush(tc.args)
		if result.Indirect != tc.indirect || tc.want == "" && len(result.Destinations) != 0 || tc.want != "" && !slices.Equal(result.Destinations, []string{tc.want}) {
			t.Fatalf("%+v: %+v", tc.args, result)
		}
	}
}

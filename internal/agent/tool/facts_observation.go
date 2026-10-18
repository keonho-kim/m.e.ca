// Purpose: Recover compact effects from committed operation facts without executing tools again.
// Pattern: Pure observation projection.
// Usage: Interrupted-work recovery applies operation records newer than its checkpoint.
// Related: internal/agent/tool/facts.go, internal/agent/tool/operation.go.
package tool

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// Observe preserves known partial mutations even when an operation ended with an error.
func (f *Facts) Observe(op Operation, output string) {
	files := map[string]FileFact{}
	for _, file := range f.Files {
		files[file.Path] = file
	}
	for path, version := range op.Versions {
		prior := files[path]
		files[path] = FileFact{Path: path, Version: version, Changed: prior.Changed}
	}
	for _, change := range op.Changes {
		if change.BeforeDigest == change.AfterDigest && change.Destination == "" {
			continue
		}
		for i := range f.Commands {
			f.Commands[i].BeforeLaterChanges = true
		}
		version := change.AfterDigest
		if change.AfterDigest == "missing" || change.Destination != "" {
			version = "missing"
		}
		if version == "" && op.Name == "create_directory" {
			version = "directory"
		}
		files[change.Path] = FileFact{Path: change.Path, Version: version, Changed: true, Stale: version == ""}
		if change.Destination != "" {
			files[change.Destination] = FileFact{Path: change.Destination, Version: change.AfterDigest, Changed: true, Stale: change.AfterDigest == ""}
		}
	}
	if op.Name == "execute_command" {
		status := "not_started"
		var exit *int
		if op.Started {
			status = "unknown"
			for p, v := range files {
				v.Stale = op.Versions[p] == ""
				files[p] = v
			}
		}
		if op.Process != nil {
			status = op.Process.Status
			n := op.Process.ExitCode
			exit = &n
		}
		command, c := ReportExcerpt(op.Command)
		text, o := ReportExcerpt(strings.TrimSpace(op.Result + "\n" + output))
		f.Commands = append(f.Commands, CommandFact{CallID: op.CallID, Command: command, Status: status, ExitCode: exit, Output: text, Excerpt: c || o, ObservedRun: op.RunID, Revision: op.Revision})
	}
	f.Files = nil
	for _, v := range files {
		f.Files = append(f.Files, v)
	}
	sort.Slice(f.Files, func(i, j int) bool { return f.Files[i].Path < f.Files[j].Path })
}

// ReportExcerpt bounds model-facing quotations; full source records remain in session storage.
func ReportExcerpt(value string) (string, bool) {
	const limit = 2048
	if len(value) <= limit {
		return value, false
	}
	end := limit
	for !utf8.RuneStart(value[end]) {
		end--
	}
	return value[:end], true
}

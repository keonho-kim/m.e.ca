// Purpose: Compare actual tool observations within one request.
// Pattern: Request-scoped observation projection.
// Usage: observeOutcome records stable source coverage, versions and diagnostic kinds.
// Related: internal/agent/progress/request.go, internal/agent/run/round_progress.go.
package run

import (
	"encoding/json"
	"fmt"
	toolapi "github.com/keonho-kim/v-code/internal/agent/tool"
	"github.com/keonho-kim/v-code/internal/policy"
)

func (s *execution) progressKey(key string) bool {
	return s.progress.Observe(key)
}

func (s *execution) toolAdvanced(call toolapi.Call, out toolapi.Result, effect policy.Effect, result string) bool {
	advanced := false
	if out.Target == "" {
		var args struct{ Path, Command string }
		if json.Unmarshal(call.Arguments, &args) == nil {
			out.Target = args.Path
			if out.Target == "" {
				out.Target = args.Command
			}
		}
	}
	for _, change := range out.Changes {
		if change.BeforeDigest != "" && change.BeforeDigest != "missing" {
			s.progressKey("worker:file:" + change.Path + ":" + change.BeforeDigest)
		}
		if change.AfterDigest == "" && change.BeforeDigest != change.AfterDigest {
			advanced = s.progressKey("worker:change:"+change.Path+":"+string(call.Arguments)) || advanced
		}
		if change.AfterDigest != "" {
			key := "worker:file:" + change.Path + ":" + change.AfterDigest
			if change.AfterDigest == "missing" && change.Destination == "" {
				key = "worker:delete:" + change.Path + ":" + change.BeforeDigest
			}
			fresh := s.progressKey(key)
			if !fresh && change.AfterDigest != "missing" && change.BeforeDigest != change.AfterDigest {
				s.round.cycle = true
			}
			advanced = fresh || advanced
		}
	}
	for path, version := range out.Versions {
		if version != "" {
			advanced = s.progressKey("worker:file:"+path+":"+version) || advanced
		}
	}
	if out.Failed || out.Failure != nil {
		if f := out.Failure; f != nil {
			target := out.Target
			if effect == policy.Command {
				target = fmt.Sprint(toolapi.FileKeys(s.checkpoint.Work.Facts().Files))
			}
			advanced = s.progressKey("worker:failure:"+call.Name+":"+target+":"+f.Code) || advanced
		}
		return advanced
	}
	if effect == policy.Command {
		if out.Process != nil && out.Process.Status == "exited" && out.Process.ExitCode != 0 {
			return advanced
		}
		// New spelling alone and repeated output variations alone are insufficient.
		// Keep actual output and status in history; this affects only stall detection.
		var p struct{ Stdout, Stderr string }
		var args struct{ Command string }
		if json.Unmarshal(out.Content, &p) == nil && json.Unmarshal(call.Arguments, &args) == nil {
			files := fmt.Sprint(toolapi.FileKeys(s.checkpoint.Work.Facts().Files))
			newCall := s.progressKey("worker:command-call:" + files + ":" + args.Command)
			newOutput := s.progressKey("worker:command-output:" + files + ":" + p.Stdout + "\x00" + p.Stderr)
			advanced = newCall && newOutput || advanced
		}
		return advanced
	}
	if len(out.Versions) == 0 {
		advanced = s.progressKey("worker:observation:"+call.Name+":"+out.Target+":"+result) || advanced
	}
	return advanced
}

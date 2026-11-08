// Purpose: Classify new shared decisions and actual execution evidence for idle recovery.
// Pattern: Bounded request-owned fingerprint window with pure evidence projection.
// Usage: Publication, execution observations and membership record stable facts under room.mu.
// Related: internal/agent/team/idle.go, internal/agent/team/observations.go.
package team

import (
	"crypto/sha256"
	"slices"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	record "github.com/keonho-kim/meca/internal/agent/record"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
)

// recordProgress retains at most 64 digests and no source strings, command IDs or outputs.
func (r *Room) recordProgress(key [32]byte) bool {
	if slices.Contains(r.progressFacts[:r.progressFactCount], key) {
		return false
	}
	r.progressFacts[r.progressFactNext] = key
	r.progressFactCount = min(len(r.progressFacts), r.progressFactCount+1)
	r.progressFactNext = (r.progressFactNext + 1) % len(r.progressFacts)
	r.progress++
	return true
}

func progressKey(parts ...string) [32]byte {
	var key [32]byte
	var joined [64]byte
	for _, part := range parts {
		copy(joined[:32], key[:])
		digest := progressPart(part)
		copy(joined[32:], digest[:])
		key = sha256.Sum256(joined[:])
	}
	return key
}

func progressPart(text string) [32]byte {
	if len(text) <= 4096 {
		var scratch [4096]byte
		n := copy(scratch[:], text)
		return sha256.Sum256(scratch[:n])
	}
	h := sha256.New()
	var scratch [4096]byte
	for len(text) > 0 {
		n := copy(scratch[:], text)
		_, _ = h.Write(scratch[:n])
		text = text[n:]
	}
	var key [32]byte
	h.Sum(key[:0])
	return key
}

func operationProgress(event record.Event) ([32]byte, bool) {
	op := event.Operation
	var key [32]byte
	found := false
	for _, change := range op.Changes {
		if change.BeforeDigest == change.AfterDigest && change.Destination == "" {
			continue
		}
		next := progressKey("file", change.Action, change.Path, change.Destination, change.AfterDigest)
		var joined [64]byte
		copy(joined[:32], key[:])
		copy(joined[32:], next[:])
		key = sha256.Sum256(joined[:])
		found = true
	}
	p := op.Process
	if p == nil {
		if op.Failure == nil {
			return key, found
		}
		errorKey := progressKey("failure", op.Name, op.Target, op.Failure.Code, op.Failure.Detail)
		var joined [64]byte
		copy(joined[:32], key[:])
		copy(joined[32:], errorKey[:])
		return sha256.Sum256(joined[:]), true
	}
	if toolapi.CommandActive(p.Status) {
		return key, found
	}
	var stdout, stderr, stdoutBytes, stderrBytes string
	if len(event.Output) > 0 {
		if root, err := sonic.GetWithOptions(event.Output, ast.SearchOptions{CopyReturn: false}); err == nil {
			stdout, _ = root.Get("Stdout").Raw()
			stderr, _ = root.Get("Stderr").Raw()
			stdoutBytes, _ = root.Get("StdoutCapturedBytes").Raw()
			stderrBytes, _ = root.Get("StderrCapturedBytes").Raw()
		}
	}
	if op.Failure == nil && op.Status != "tool_failed" && len(stdout) <= 2 && len(stderr) <= 2 && p.Diagnostic == "" && p.Status == "exited" && p.ExitCode == 0 {
		return key, found
	}
	failure, detail := "", ""
	if op.Failure != nil {
		failure = op.Failure.Code
		detail = op.Failure.Detail
	}
	command := progressKey("command", op.Command, p.Status, strconv.Itoa(p.ExitCode), stdout, stderr, stdoutBytes, stderrBytes, p.Diagnostic, failure, detail)
	var joined [64]byte
	copy(joined[:32], key[:])
	copy(joined[32:], command[:])
	return sha256.Sum256(joined[:]), true
}

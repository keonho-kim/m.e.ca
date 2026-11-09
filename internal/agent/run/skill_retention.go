// Purpose: Retain only current skill and reference versions under one request byte limit.
// Pattern: Request-owned instruction evidence index.
// Usage: Explicit reads and successful native reads update this index after persistence.
// Related: internal/agent/run/skills.go, internal/agent/context/skill_reads.go.
package run

import (
	"path/filepath"
	"strconv"
	"strings"

	modelapi "github.com/keonho-kim/meca/internal/agent/model"
	toolapi "github.com/keonho-kim/meca/internal/agent/tool"
	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

const maxSkillReadBytes = 8 << 20

func (s *execution) allowExplicitSkill(id string) {
	for _, existing := range s.explicitSkills {
		if existing == id {
			return
		}
	}
	s.explicitSkills = append(s.explicitSkills, id)
}

func referenceKey(read instruction.ReferenceRead) string {
	return "file:" + read.ID + "\x00" + filepath.Clean(read.Path) + "\x00"
}

func (s *execution) checkInstruction(key, discardPrefix string, message modelapi.Message) error {
	bytes := s.skillReadBytes + message.ContextBytes()
	for i, old := range s.skillReadKeys {
		if old == key || discardPrefix != "" && strings.HasPrefix(old, discardPrefix) {
			bytes -= s.skillReads[i].ContextBytes()
		}
	}
	if bytes > maxSkillReadBytes {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.SkillsInstructionBudget)
	}
	return nil
}

func (s *execution) checkSkillResult(call toolapi.Call, result toolapi.Result) error {
	message := modelapi.Message{Content: s.runner.Redact(string(result.Content)), ToolCallID: call.ID, ToolName: call.Name}
	if result.Skill != nil {
		return s.checkInstruction("skill:"+result.Skill.ID, "", message)
	}
	if read := result.SkillFile; read != nil {
		base, discard := referenceKey(*read), ""
		if old := s.skillFileVersions[base]; old != "" && old != read.Revision {
			discard = base
		}
		return s.checkInstruction(base+strconv.FormatInt(read.Offset, 10), discard, message)
	}
	return nil
}

func (s *execution) retainInstruction(key, revision string, message modelapi.Message) error {
	if s.skillReadIndex == nil {
		s.skillReadIndex = make(map[string]int)
		s.skillReadRevisions = make(map[string]string)
	}
	previous := 0
	if index, ok := s.skillReadIndex[key]; ok {
		previous = s.skillReads[index].ContextBytes()
	}
	if s.skillReadBytes-previous+message.ContextBytes() > maxSkillReadBytes {
		return faults.New(diagcatalog.ResourceExhausted, diagcatalog.SkillsInstructionBudget)
	}
	s.skillReadBytes += message.ContextBytes() - previous
	s.skillReadRevisions[key] = revision
	if index, ok := s.skillReadIndex[key]; ok {
		s.skillReads[index] = message
		return nil
	}
	s.skillReadIndex[key] = len(s.skillReads)
	s.skillReadKeys = append(s.skillReadKeys, key)
	s.skillReads = append(s.skillReads, message)
	return nil
}

func (s *execution) retainSkillRead(doc instruction.Document, message modelapi.Message) error {
	return s.retainInstruction("skill:"+doc.ID, doc.Revision, message)
}

func (s *execution) retainSkillReference(read instruction.ReferenceRead, message modelapi.Message) error {
	base := referenceKey(read)
	if s.skillFileVersions == nil {
		s.skillFileVersions = make(map[string]string)
	}
	if previous := s.skillFileVersions[base]; previous != "" && previous != read.Revision {
		s.forgetInstructions(base)
	}
	s.skillFileVersions[base] = read.Revision
	return s.retainInstruction(base+strconv.FormatInt(read.Offset, 10), read.Revision, message)
}

func (s *execution) forgetInstructions(prefix string) {
	old, keys := s.skillReads, s.skillReadKeys
	kept, keptKeys := old[:0], keys[:0]
	for i, key := range keys {
		if strings.HasPrefix(key, prefix) {
			s.skillReadBytes -= old[i].ContextBytes()
			delete(s.skillReadIndex, key)
			delete(s.skillReadRevisions, key)
			continue
		}
		s.skillReadIndex[key] = len(kept)
		kept = append(kept, old[i])
		keptKeys = append(keptKeys, key)
	}
	clear(old[len(kept):])
	clear(keys[len(keptKeys):])
	s.skillReads, s.skillReadKeys = kept, keptKeys
}

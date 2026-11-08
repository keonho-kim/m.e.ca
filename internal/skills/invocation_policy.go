// Purpose: Read optional agent invocation policy without mixing it with SKILL.md document construction.
// Pattern: Confined metadata I/O adapter.
// Usage: documentAt applies this policy before publishing skill metadata or its revision.
// Related: internal/skills/files.go, internal/skills/format.go, internal/skills/instruction/document.go.
package skills

import (
	"errors"
	"io"
	"os"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
)

func readInvocationPolicy(root *os.Root, metadata *instruction.Metadata) (err error) {
	file, err := openRegular(root, "agents/openai.yaml", MaxMetadataBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	body, err := io.ReadAll(io.LimitReader(file, MaxMetadataBytes+1))
	if err != nil {
		return err
	}
	if len(body) > MaxMetadataBytes {
		return ErrLimit
	}
	var policy struct {
		Policy struct {
			Allow *bool `yaml:"allow_implicit_invocation"`
		} `yaml:"policy"`
	}
	node, err := metadataNode(body)
	if err != nil {
		return err
	}
	if err := node.Decode(&policy); err != nil {
		return faults.Errorf(diagcatalog.SkillsMalformedAgentsOpenaiYaml, ErrInvalid)
	}
	if policy.Policy.Allow != nil && !*policy.Policy.Allow {
		metadata.ManualOnly = true
	}
	return nil
}

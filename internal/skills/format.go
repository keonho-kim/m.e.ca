// Purpose: Parse portable SKILL.md metadata with strict bounded YAML structure.
// Pattern: Pure format boundary.
// Usage: Discovery reads frontmatter; full reads reuse the same parser.
// Related: internal/skills/files.go, internal/skills/reader.go.
package skills

import (
	"bytes"
	"strings"
	"unicode/utf8"

	diagcatalog "github.com/keonho-kim/meca/internal/diagnostics/catalog"
	faults "github.com/keonho-kim/meca/internal/diagnostics/fault"
	"github.com/keonho-kim/meca/internal/skills/instruction"
	"go.yaml.in/yaml/v3"
)

type manifest struct {
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
	Compatibility string `yaml:"compatibility"`
	ManualOnly    bool   `yaml:"disable-model-invocation"`
}

func parseMetadata(body []byte, folder string) (instruction.Metadata, error) {
	node, err := metadataNode(body)
	if err != nil {
		return instruction.Metadata{}, err
	}
	var value manifest
	if err := node.Decode(&value); err != nil {
		return instruction.Metadata{}, faults.Errorf(diagcatalog.SkillsDuplicateKeysOrInvalidMetadataTypes, ErrInvalid)
	}
	fields := node.Content[0].Content
	var warnings []string
	for i := 0; i < len(fields); i += 2 {
		key, entry := fields[i].Value, fields[i+1]
		if key == "name" || key == "description" || key == "compatibility" {
			if entry.Tag != "!!str" {
				return instruction.Metadata{}, faults.Errorf(diagcatalog.SkillsTextMetadataMustUseStrings, ErrInvalid)
			}
		}
		if key == "hooks" || key == "context" || key == "allowed-tools" {
			warnings = append(warnings, diagcatalog.Text(diagcatalog.SkillsExtensionIgnored, key))
		}
	}
	if !instruction.ValidName(value.Name) || value.Name != folder || strings.TrimSpace(value.Description) == "" || utf8.RuneCountInString(value.Description) > 1024 || utf8.RuneCountInString(value.Compatibility) > 500 {
		return instruction.Metadata{}, faults.Errorf(diagcatalog.SkillsNameDescriptionOrCompatibilityViolatesTheSkill, ErrInvalid)
	}
	return instruction.Metadata{Name: value.Name, Description: value.Description, Compatibility: value.Compatibility, ManualOnly: value.ManualOnly, Warnings: warnings}, nil
}

func metadataNode(body []byte) (*yaml.Node, error) {
	if !utf8.Valid(body) || bytes.IndexByte(body, 0) >= 0 {
		return nil, faults.Errorf(diagcatalog.SkillsMetadataMustBeUTFText, ErrInvalid)
	}
	var node yaml.Node
	if err := yaml.Unmarshal(body, &node); err != nil {
		return nil, faults.Errorf(diagcatalog.SkillsMalformedYAML, ErrInvalid)
	}
	nodes := 0
	var check func(*yaml.Node, int) error
	check = func(n *yaml.Node, depth int) error {
		nodes++
		if depth > 32 || nodes > 4096 || n.Kind == yaml.AliasNode {
			return faults.Errorf(diagcatalog.SkillsExcessiveYAMLStructureOrAlias, ErrInvalid)
		}
		if n.Kind == yaml.MappingNode {
			keys := make(map[string]bool)
			for i := 0; i < len(n.Content); i += 2 {
				key := n.Content[i]
				if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || keys[key.Value] {
					return faults.Errorf(diagcatalog.SkillsYAMLKeysMustBeUniqueStrings, ErrInvalid)
				}
				keys[key.Value] = true
			}
		}
		for _, child := range n.Content {
			if err := check(child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(&node, 0); err != nil {
		return nil, err
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return nil, faults.Errorf(diagcatalog.SkillsMetadataMustBeAMapping, ErrInvalid)
	}
	return &node, nil
}

func frontmatter(body []byte) ([]byte, int, error) {
	first, rest, ok := bytes.Cut(body, []byte("\n"))
	if !ok || string(bytes.TrimSuffix(first, []byte("\r"))) != "---" {
		return nil, 0, faults.Errorf(diagcatalog.SkillsSKILLMdRequiresYAMLFrontmatter, ErrInvalid)
	}
	start := len(body) - len(rest)
	offset := start
	for len(rest) > 0 {
		line, tail, newline := bytes.Cut(rest, []byte("\n"))
		if string(bytes.TrimSuffix(line, []byte("\r"))) == "---" {
			end := offset + len(line)
			if newline {
				end++
			}
			return body[start:offset], end, nil
		}
		offset += len(line) + 1
		rest = tail
	}
	return nil, 0, faults.Errorf(diagcatalog.SkillsMissingOrOversizedFrontmatterTerminator, ErrInvalid)
}

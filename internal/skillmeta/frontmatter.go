// Package skillmeta validates the YAML frontmatter of the skills/ registry so
// that every SKILL.md is discoverable and structurally sound.
package skillmeta

import (
	"bytes"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// Frontmatter is the metadata block every SKILL.md carries.
type Frontmatter struct {
	Name         string      `yaml:"name"`
	Version      string      `yaml:"version"`
	Description  string      `yaml:"description"`
	Metadata     Metadata    `yaml:"metadata"`
	AgentCreated interface{} `yaml:"agent_created"`
	License      interface{} `yaml:"license"`
}

// Metadata holds the nested metadata fields of a skill.
type Metadata struct {
	Requires     Requires    `yaml:"requires"`
	CLIHelp      string      `yaml:"cliHelp"`
	Orchestrates []string    `yaml:"orchestrates"`
	Scenario     interface{} `yaml:"scenario"`
	Platforms    interface{} `yaml:"platforms"`
	Shortcuts    interface{} `yaml:"shortcuts"`
}

// Requires lists what a skill needs to run.
type Requires struct {
	Bins         []string    `yaml:"bins"`
	OptionalBins []string    `yaml:"optional_bins"`
	BinsAny      []string    `yaml:"bins_any"`
	BinsNote     interface{} `yaml:"bins_note"`
	Python       interface{} `yaml:"python"`
}

var errNoFrontmatter = errors.New("no `---` delimited frontmatter block")

// ExtractFrontmatter returns the YAML between the first pair of `---` fences.
func ExtractFrontmatter(src []byte) ([]byte, error) {
	const fence = "---"
	trimmed := bytes.TrimLeft(src, " \t\r\n")
	if !bytes.HasPrefix(trimmed, []byte(fence)) {
		return nil, errNoFrontmatter
	}
	rest := trimmed[len(fence):]
	idx := bytes.Index(rest, []byte("\n"+fence))
	if idx < 0 {
		return nil, errNoFrontmatter
	}
	return rest[:idx], nil
}

// ParseFrontmatter strictly decodes the frontmatter, rejecting unknown or
// wrongly-nested keys so that structural mistakes surface as errors instead of
// being silently dropped.
func ParseFrontmatter(block []byte) (Frontmatter, error) {
	var fm Frontmatter
	dec := yaml.NewDecoder(bytes.NewReader(block))
	dec.KnownFields(true)
	if err := dec.Decode(&fm); err != nil {
		return Frontmatter{}, fmt.Errorf("invalid frontmatter: %w", err)
	}
	return fm, nil
}

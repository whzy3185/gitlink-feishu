package skillmeta

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

// Problem is a single validation failure against the shared skill schema.
type Problem struct {
	Skill   string
	Field   string
	Message string
}

func (p Problem) String() string {
	return fmt.Sprintf("%s: %s: %s", p.Skill, p.Field, p.Message)
}

var (
	semverRe   = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	skillDirRe = regexp.MustCompile(`^gitlink-`)
)

const minDescriptionRunes = 20

// Validate checks every <root>/gitlink-*/SKILL.md against the shared schema and
// returns all problems found; it reports the whole registry rather than
// stopping at the first failure.
func Validate(root string) ([]Problem, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var problems []Problem
	for _, entry := range entries {
		if !entry.IsDir() || !skillDirRe.MatchString(entry.Name()) {
			continue
		}
		problems = append(problems, validateSkill(root, entry.Name())...)
	}
	sort.Slice(problems, func(i, j int) bool {
		if problems[i].Skill != problems[j].Skill {
			return problems[i].Skill < problems[j].Skill
		}
		return problems[i].Field < problems[j].Field
	})
	return problems, nil
}

func validateSkill(root, name string) []Problem {
	var ps []Problem
	add := func(field, msg string) {
		ps = append(ps, Problem{Skill: name, Field: field, Message: msg})
	}

	src, err := os.ReadFile(filepath.Join(root, name, "SKILL.md"))
	if err != nil {
		add("SKILL.md", "cannot read: "+err.Error())
		return ps
	}
	block, err := ExtractFrontmatter(src)
	if err != nil {
		add("frontmatter", err.Error())
		return ps
	}
	fm, err := ParseFrontmatter(block)
	if err != nil {
		add("frontmatter", err.Error())
		return ps
	}

	switch {
	case fm.Name == "":
		add("name", "must not be empty")
	case fm.Name != name:
		add("name", fmt.Sprintf("must equal the directory name %q", name))
	}
	if !semverRe.MatchString(fm.Version) {
		add("version", "must be semantic version X.Y.Z")
	}
	if utf8.RuneCountInString(fm.Description) < minDescriptionRunes {
		add("description", fmt.Sprintf("must be at least %d characters; it is the router's only routing signal", minDescriptionRunes))
	}
	if !containsString(fm.Metadata.Requires.Bins, "gitlink-cli") && !containsString(fm.Metadata.Requires.BinsAny, "gitlink-cli") {
		add("metadata.requires.bins", `must contain "gitlink-cli"`)
	}
	if strings.TrimSpace(fm.Metadata.CLIHelp) == "" {
		add("metadata.cliHelp", "must name the command group, e.g. \"gitlink-cli x --help\"")
	}
	return ps
}

func containsString(xs []string, target string) bool {
	for _, x := range xs {
		if x == target {
			return true
		}
	}
	return false
}

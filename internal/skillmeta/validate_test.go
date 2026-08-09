package skillmeta

import "testing"

// TestRepoSkillsValid treats the real skills/ registry as a regression
// baseline: once fixed, every SKILL.md must keep passing the schema.
func TestRepoSkillsValid(t *testing.T) {
	problems, err := Validate("../../skills")
	if err != nil {
		t.Fatalf("validate skills: %v", err)
	}
	for _, p := range problems {
		t.Errorf("unexpected problem in registry: %s", p)
	}
}

// TestValidateCatchesBadSkills pins each rule to a deliberately broken sample.
func TestValidateCatchesBadSkills(t *testing.T) {
	problems, err := Validate("testdata/bad")
	if err != nil {
		t.Fatalf("validate testdata: %v", err)
	}
	got := make(map[string]bool)
	for _, p := range problems {
		got[p.Skill+"/"+p.Field] = true
	}
	want := []string{
		"gitlink-noversion/version",
		"gitlink-nobins/metadata.requires.bins",
		"gitlink-nohelp/metadata.cliHelp",
		"gitlink-flat/frontmatter",
		"gitlink-mismatch/name",
	}
	for _, w := range want {
		if !got[w] {
			t.Errorf("expected problem %q, got %v", w, problems)
		}
	}
}

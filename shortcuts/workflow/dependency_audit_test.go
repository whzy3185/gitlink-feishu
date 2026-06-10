package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseGoModForDependencyAudit(t *testing.T) {
	input, err := ParseGoModForDependencyAudit(`module example.com/project

go 1.21
toolchain go1.22.1

require (
	github.com/acme/stable v1.2.3
	github.com/acme/pseudo v0.0.0-20240501120000-abcdef123456 // indirect
	github.com/acme/major v2.1.0
)

replace github.com/acme/local => ../local
replace github.com/acme/fork v1.0.0 => github.com/fork/acme v1.0.1
`)
	if err != nil {
		t.Fatalf("ParseGoModForDependencyAudit returned error: %v", err)
	}
	if input.Module != "example.com/project" || input.GoVersion != "1.21" || input.Toolchain != "go1.22.1" {
		t.Fatalf("module/go/toolchain = %q/%q/%q", input.Module, input.GoVersion, input.Toolchain)
	}
	if len(input.Requirements) != 3 || !input.Requirements[1].Indirect {
		t.Fatalf("requirements = %+v, want 3 with second indirect", input.Requirements)
	}
	if len(input.Replacements) != 2 || input.Replacements[0].NewPath != "../local" {
		t.Fatalf("replacements = %+v, want local replacement", input.Replacements)
	}
}

func TestAnalyzeDependencyAuditFindsRiskSignals(t *testing.T) {
	result := AnalyzeDependencyAudit(DependencyAuditInput{
		Repository: "owner/repo",
		Module:     "example.com/project",
		GoVersion:  "1.19",
		Requirements: []DependencyRequirement{
			{Path: "github.com/acme/pseudo", Version: "v0.0.0-20240501120000-abcdef123456", Indirect: true},
			{Path: "github.com/acme/major", Version: "v2.1.0"},
			{Path: "github.com/acme/beta/v3", Version: "v3.0.0-rc.1"},
		},
		Replacements: []DependencyReplacement{
			{OldPath: "github.com/acme/local", NewPath: "./local"},
		},
		Source: "go.mod",
	}, "en")

	if result.RiskLevel != "high" {
		t.Fatalf("RiskLevel = %q, want high", result.RiskLevel)
	}
	if result.DirectDependencies != 2 || result.IndirectDependencies != 1 {
		t.Fatalf("direct/indirect = %d/%d, want 2/1", result.DirectDependencies, result.IndirectDependencies)
	}
	if result.PseudoVersionCount != 1 || result.LocalReplacementCount != 1 || result.MajorMismatchCount != 1 || result.PreReleaseCount != 1 {
		t.Fatalf("counts = pseudo:%d local:%d major:%d pre:%d", result.PseudoVersionCount, result.LocalReplacementCount, result.MajorMismatchCount, result.PreReleaseCount)
	}
	if !hasDependencyFinding(result.Findings, "local_replace") || !hasDependencyFinding(result.Findings, "major_version_mismatch") {
		t.Fatalf("findings missing expected codes: %+v", result.Findings)
	}
	if result.Score >= 100 || len(result.Recommendations) == 0 {
		t.Fatalf("score/recommendations = %d/%v, want risk penalty and recommendations", result.Score, result.Recommendations)
	}
}

func TestReadDependencyAuditInputSupportsJSONAndGoMod(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "dependency_audit.json")
	goModPath := filepath.Join(dir, "go.mod")
	writeJSONFixture(t, jsonPath, DependencyAuditInput{
		Module: "example.com/json",
		Requirements: []DependencyRequirement{
			{Path: "github.com/acme/lib", Version: "v1.0.0"},
		},
	})
	if err := writeTextFixture(goModPath, "module example.com/mod\n\ngo 1.21\nrequire github.com/acme/lib v1.0.0\n"); err != nil {
		t.Fatalf("write go.mod fixture: %v", err)
	}

	jsonInput, err := readDependencyAuditInput(jsonPath)
	if err != nil {
		t.Fatalf("readDependencyAuditInput(JSON) returned error: %v", err)
	}
	if jsonInput.Module != "example.com/json" || jsonInput.Source != jsonPath {
		t.Fatalf("json input = %+v", jsonInput)
	}

	goModInput, err := readDependencyAuditInput(goModPath)
	if err != nil {
		t.Fatalf("readDependencyAuditInput(go.mod) returned error: %v", err)
	}
	if goModInput.Module != "example.com/mod" || len(goModInput.Requirements) != 1 {
		t.Fatalf("go.mod input = %+v", goModInput)
	}
}

func TestRenderDependencyAuditMarkdownAndTable(t *testing.T) {
	result := AnalyzeDependencyAudit(DependencyAuditInput{
		Module:    "example.com/project",
		GoVersion: "1.21",
		Requirements: []DependencyRequirement{
			{Path: "github.com/acme/lib", Version: "v1.0.0"},
		},
	}, "zh-CN")

	markdown, err := RenderDependencyAudit(result, "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderDependencyAudit markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# 依赖风险审计") || !strings.Contains(markdown, "example.com/project") {
		t.Fatalf("markdown output missing expected content:\n%s", markdown)
	}

	table, err := RenderDependencyAudit(result, "table", "en")
	if err != nil {
		t.Fatalf("RenderDependencyAudit table returned error: %v", err)
	}
	if !strings.Contains(table, "MODULE") || !strings.Contains(table, "example.com/project") {
		t.Fatalf("table output missing expected content:\n%s", table)
	}
}

func TestShortcutsExposeDependencyAudit(t *testing.T) {
	names := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		names[shortcut.Name] = true
	}
	if !names["dependency-audit"] {
		t.Fatal("Shortcuts missing dependency-audit")
	}
}

func hasDependencyFinding(findings []DependencyAuditFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func writeTextFixture(path, content string) error {
	return os.WriteFile(path, []byte(content), 0600)
}

package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func sampleConfig() ProjectConfig {
	create := true
	return ProjectConfig{
		Project: ProjectInfo{
			Name:        "Demo Project",
			Description: "Demo description",
			Language:    "Go",
			License:     "MulanPSL-2.0",
		},
		Repository: RepositoryInfo{Owner: "alice", Name: "demo"},
		Branches:   []BranchPlan{{Name: "develop", From: "master", Create: &create}},
		Issues: []IssuePlan{
			{
				Title:    "Write README",
				Type:     "documentation",
				Priority: "normal",
				Tasks:    []string{"Add quickstart", "Add license"},
			},
		},
	}
}

func TestPlannedFilesIncludeRequiredProjectArtifacts(t *testing.T) {
	files := plannedFiles(sampleConfig())
	for _, path := range []string{"README.md", "LICENSE", ".github/workflows/ci.yml", "docs/CONTRIBUTING.md"} {
		if _, ok := files[path]; !ok {
			t.Fatalf("expected planned file %s", path)
		}
	}
}

func TestBuildCLIPlanChainsMoreThanThreeGitlinkCommands(t *testing.T) {
	plan := buildCLIPlan(sampleConfig(), "", false)
	if len(plan) < 4 {
		t.Fatalf("expected at least 4 commands, got %d", len(plan))
	}
	if strings.Join(plan[0][:2], " ") != "repo +info" {
		t.Fatalf("unexpected first command: %#v", plan[0])
	}
	if !containsCommand(plan, "branch +list") {
		t.Fatalf("branch +list command missing: %#v", plan)
	}
	if !containsCommand(plan, "issue +create") {
		t.Fatalf("issue +create command missing: %#v", plan)
	}
}

func TestBuildCLIPlanCanCreateRepositoryFirst(t *testing.T) {
	plan := buildCLIPlan(sampleConfig(), "", true)
	if strings.Join(plan[0][:2], " ") != "repo +create" {
		t.Fatalf("unexpected first command: %#v", plan[0])
	}
	if strings.Join(plan[1][:2], " ") != "repo +info" {
		t.Fatalf("unexpected second command: %#v", plan[1])
	}
}

func TestIssueBodyContainsChecklistAndRepository(t *testing.T) {
	config := sampleConfig()
	body := issueBody(config.Issues[0], config)
	if !strings.Contains(body, "`alice/demo`") {
		t.Fatalf("repository missing from body: %s", body)
	}
	if !strings.Contains(body, "- [ ] Add quickstart") {
		t.Fatalf("checklist missing from body: %s", body)
	}
}

func TestWriteOutputsCreatesReportManifestAndSummary(t *testing.T) {
	tmp := t.TempDir()
	paths, err := writeOutputs(sampleConfig(), tmp, time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("writeOutputs returned error: %v", err)
	}
	for _, path := range []string{paths.Report, paths.Summary, paths.Manifest, paths.Files} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected output %s: %v", path, err)
		}
	}
	data, err := os.ReadFile(paths.Manifest)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var manifest OutputManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}
	if manifest.Repository != "alice/demo" {
		t.Fatalf("unexpected repository: %s", manifest.Repository)
	}
	if len(manifest.Files) < 4 {
		t.Fatalf("expected at least 4 files, got %d", len(manifest.Files))
	}
	if filepath.Base(paths.Report) == "" {
		t.Fatal("report path should include filename")
	}
}

func TestBranchAlreadyExistsIsIdempotentSkip(t *testing.T) {
	if !isIdempotentSkip("[-1] 新分支已存在！") {
		t.Fatal("expected existing branch error to be skipped")
	}
	if isIdempotentSkip("[401] 请登录后再操作") {
		t.Fatal("auth error should not be skipped")
	}
}

func containsCommand(plan [][]string, command string) bool {
	for _, item := range plan {
		if len(item) >= 2 && strings.Join(item[:2], " ") == command {
			return true
		}
	}
	return false
}

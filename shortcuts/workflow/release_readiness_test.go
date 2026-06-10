package workflow

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestAnalyzeReleaseReadinessPassesCleanRelease(t *testing.T) {
	result := AnalyzeReleaseReadiness(ReleaseReadinessInput{
		Repository:     "owner/repo",
		Version:        "v1.2.0",
		Changes:        []string{"Add workflow release gate"},
		Tests:          []ReleaseCheckItem{{Name: "go test ./...", Status: "passed"}},
		Artifacts:      []ReleaseCheckItem{{Name: "windows zip", Status: "passed"}},
		RollbackPlan:   "Revert tag and restore previous release assets.",
		DependencyRisk: "clean",
	}, "en")

	if result.Gate != ReleaseGatePass {
		t.Fatalf("Gate = %q, want pass: %+v", result.Gate, result.Findings)
	}
	if result.Score != 100 || result.PassedChecks != 2 || len(result.Findings) != 0 {
		t.Fatalf("result = %+v, want clean score", result)
	}
}

func TestAnalyzeReleaseReadinessBlocksKnownFailures(t *testing.T) {
	result := AnalyzeReleaseReadiness(ReleaseReadinessInput{
		Version:         "v1.2.0",
		Changes:         []string{"Breaking config migration"},
		BreakingChanges: true,
		KnownBlockers:   []string{"Windows artifact is not signed"},
		Tests:           []ReleaseCheckItem{{Name: "go test ./...", Status: "failed"}},
		Artifacts:       []ReleaseCheckItem{{Name: "linux tar", Status: "missing"}},
		DependencyRisk:  "high",
	}, "en")

	if result.Gate != ReleaseGateBlock {
		t.Fatalf("Gate = %q, want block", result.Gate)
	}
	if result.FailedChecks != 1 || result.MissingChecks != 1 || result.Score >= 60 {
		t.Fatalf("failed/missing/score = %d/%d/%d", result.FailedChecks, result.MissingChecks, result.Score)
	}
	if !hasReleaseFinding(result.Findings, "known_blocker") || !hasReleaseFinding(result.Findings, "missing_rollback_plan") {
		t.Fatalf("findings missing expected blockers: %+v", result.Findings)
	}
}

func TestCollectReleaseReadinessInputFromFlags(t *testing.T) {
	ctx := &common.RuntimeContext{
		Args: map[string]string{
			"repository":       "owner/repo",
			"version":          "v1.0.0",
			"changes":          "feature A,fix B",
			"breaking-changes": "true",
			"tests":            "go test ./...=passed,go build ./...=passed",
			"artifacts":        "windows zip=missing",
			"rollback-plan":    "Revert tag",
			"dependency-risk":  "medium",
		},
	}
	input, err := collectReleaseReadinessInput(ctx)
	if err != nil {
		t.Fatalf("collectReleaseReadinessInput returned error: %v", err)
	}
	if input.Repository != "owner/repo" || input.Version != "v1.0.0" || !input.BreakingChanges {
		t.Fatalf("input = %+v", input)
	}
	if len(input.Changes) != 2 || len(input.Tests) != 2 || input.Artifacts[0].Status != "missing" {
		t.Fatalf("parsed lists = changes:%v tests:%v artifacts:%v", input.Changes, input.Tests, input.Artifacts)
	}
}

func TestReadReleaseReadinessInputFromJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "release_readiness.json")
	writeJSONFixture(t, path, ReleaseReadinessInput{
		Repository: "owner/repo",
		Version:    "v2.0.0",
		Changes:    []string{"Major release"},
		Tests:      []ReleaseCheckItem{{Name: "go test ./...", Status: "passed"}},
		Artifacts:  []ReleaseCheckItem{{Name: "linux tar", Status: "passed"}},
	})

	input, err := readReleaseReadinessInput(path)
	if err != nil {
		t.Fatalf("readReleaseReadinessInput returned error: %v", err)
	}
	if input.Version != "v2.0.0" || len(input.Tests) != 1 {
		t.Fatalf("input = %+v", input)
	}
}

func TestRenderReleaseReadinessMarkdownAndTable(t *testing.T) {
	result := AnalyzeReleaseReadiness(ReleaseReadinessInput{
		Version:   "v1.0.0",
		Changes:   []string{"Initial release"},
		Tests:     []ReleaseCheckItem{{Name: "go test ./...", Status: "passed"}},
		Artifacts: []ReleaseCheckItem{{Name: "linux tar", Status: "passed"}},
	}, "zh-CN")

	markdown, err := RenderReleaseReadiness(result, "markdown", "zh-CN")
	if err != nil {
		t.Fatalf("RenderReleaseReadiness markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# 发布就绪度检查") || !strings.Contains(markdown, "v1.0.0") {
		t.Fatalf("markdown output missing expected content:\n%s", markdown)
	}

	table, err := RenderReleaseReadiness(result, "table", "en")
	if err != nil {
		t.Fatalf("RenderReleaseReadiness table returned error: %v", err)
	}
	if !strings.Contains(table, "VERSION") || !strings.Contains(table, "pass") {
		t.Fatalf("table output missing expected content:\n%s", table)
	}
}

func TestShortcutsExposeReleaseReadiness(t *testing.T) {
	names := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		names[shortcut.Name] = true
	}
	if !names["release-readiness"] {
		t.Fatal("Shortcuts missing release-readiness")
	}
}

func hasReleaseFinding(findings []ReleaseReadinessFinding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

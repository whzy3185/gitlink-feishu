package workflow

import (
	"testing"
	"time"
)

func TestBuildReviewAgentPlanRoutesIncrementalScopesAndKeepsOwnerDecision(t *testing.T) {
	previous := ReviewContext{
		Repository:     "owner/repo",
		PullRequest:    42,
		CurrentHeadSHA: "head-1",
		Files: []map[string]interface{}{
			{"filename": "internal/auth/token.go", "sha": "old"},
			{"filename": "README.md", "sha": "same"},
		},
		WorkItem: ReviewWorkItem{SourceFingerprint: "fingerprint-1"},
	}
	current := previous
	current.CurrentHeadSHA = "head-2"
	current.WorkItem.SourceFingerprint = "fingerprint-2"
	current.Files = []map[string]interface{}{
		{"filename": "internal/auth/token.go", "sha": "new"},
		{"filename": "README.md", "sha": "same"},
		{"filename": "internal/auth/token_test.go", "sha": "added"},
	}
	plan := BuildReviewAgentPlan(
		current,
		&previous,
		3,
		time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC),
	)
	if plan.Mode != "incremental" || plan.GitLinkWrites != 0 || !plan.HumanDecisionRequired {
		t.Fatalf("plan boundary = %#v", plan)
	}
	if len(plan.ChangedFiles) != 2 {
		t.Fatalf("changed files = %#v", plan.ChangedFiles)
	}
	roles := map[string]bool{}
	for _, task := range plan.Tasks {
		roles[task.Role] = true
		if !task.ReadOnly {
			t.Fatalf("task is not read-only: %#v", task)
		}
	}
	for _, role := range []string{"correctness", "tests", "security", "integration"} {
		if !roles[role] {
			t.Fatalf("missing role %s: %#v", role, roles)
		}
	}
	for _, capability := range plan.Capabilities {
		if capability.Action == "merge" && capability.Enabled {
			t.Fatal("merge must remain disabled")
		}
	}
}

func TestSynthesizeReviewAssessmentsRejectsStaleAndNeverApproves(t *testing.T) {
	plan := ReviewAgentPlan{
		SchemaVersion: reviewAgentPlanSchema,
		RunID:         "run-1",
		HeadSHA:       "head-2",
		Tasks: []ReviewAgentTask{
			{TaskID: "task-correctness", Role: "correctness"},
			{TaskID: "task-tests", Role: "tests"},
		},
	}
	assessments := []ReviewAgentAssessment{
		{
			SchemaVersion: reviewAgentAssessmentSchema,
			RunID:         plan.RunID,
			TaskID:        "task-correctness",
			Role:          "correctness",
			HeadSHA:       plan.HeadSHA,
			Findings: []ReviewAgentFinding{{
				Severity:   "high",
				Path:       "main.go",
				Summary:    "state can be lost",
				Evidence:   "write is not persisted before acknowledgement",
				Confidence: "high",
			}},
		},
		{
			SchemaVersion: reviewAgentAssessmentSchema,
			RunID:         plan.RunID,
			TaskID:        "task-tests",
			Role:          "tests",
			HeadSHA:       "stale-head",
		},
	}
	result := SynthesizeReviewAssessments(plan, assessments)
	if result.Status != "incomplete" || len(result.Conflicts) == 0 ||
		result.Recommendation != "human_decision_required" ||
		!result.HumanDecisionRequired || result.GitLinkWrites != 0 {
		t.Fatalf("synthesis = %#v", result)
	}
}

func TestBuildReviewWarroomKeepsCrossRepositoryFactsAndPartialCounts(t *testing.T) {
	contexts := []ReviewContext{
		{
			Repository:       "owner/one",
			PullRequest:      1,
			CurrentHeadSHA:   "head-1",
			CollectionStatus: "complete",
			WorkItem: ReviewWorkItem{
				PRKey:        "owner/one#1",
				Priority:     "low",
				ReviewStage:  "human_reviewing",
				GitLinkState: "open",
				GeneratedAt:  "2026-07-31T08:00:00Z",
			},
		},
		{
			Repository:       "owner/two",
			PullRequest:      2,
			CurrentHeadSHA:   "head-2",
			CollectionStatus: "partial",
			Partial:          true,
			WorkItem: ReviewWorkItem{
				PRKey:        "owner/two#2",
				Priority:     "high",
				ReviewStage:  "waiting_for_re_review",
				GitLinkState: "open",
				GeneratedAt:  "2026-07-31T08:00:00Z",
			},
		},
	}
	warroom := BuildReviewWarroom(contexts, time.Now())
	if len(warroom.Repositories) != 2 || warroom.CompleteItems != 1 ||
		warroom.PartialItems != 1 || warroom.Items[0].PRKey != "owner/two#2" ||
		warroom.GitLinkWrites != 0 {
		t.Fatalf("warroom = %#v", warroom)
	}
}

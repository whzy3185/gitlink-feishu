package workflow

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/collab"
)

func TestRunReviewAgentPlanUsesBoundedValidatedHTTPContract(t *testing.T) {
	t.Setenv("AGENT_TEST_TOKEN", "agent-secret")
	var mu sync.Mutex
	invocations := []ReviewAgentInvocation{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer agent-secret" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var invocation ReviewAgentInvocation
		if err := json.NewDecoder(request.Body).Decode(&invocation); err != nil {
			t.Fatalf("decode invocation: %v", err)
		}
		mu.Lock()
		invocations = append(invocations, invocation)
		mu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(ReviewAgentAssessment{
			SchemaVersion: reviewAgentAssessmentSchema,
			RunID:         invocation.RunID, TaskID: invocation.Task.TaskID,
			Role: invocation.Task.Role, HeadSHA: invocation.HeadSHA,
			Status: "completed", AssessmentSummary: "Reviewed assigned scope.",
			Coverage:        []string{"assigned_file_scopes"},
			EvidenceChecked: []string{"review_context", "changed_files"},
			Unknowns:        []string{"No blocking finding in the supplied evidence."},
			CompletedAt:     "2026-08-01T15:00:00Z",
		})
	}))
	defer server.Close()
	provider, err := NewHTTPReviewAgentProvider(server.URL, "env:AGENT_TEST_TOKEN", server.Client())
	if err != nil {
		t.Fatalf("NewHTTPReviewAgentProvider: %v", err)
	}
	plan := ReviewAgentPlan{
		SchemaVersion: reviewAgentPlanSchema, RunID: "run-agent-42",
		Repository: "owner/repo", PRNumber: 42, HeadSHA: "head-42",
		SourceFingerprint: "fingerprint-42", Mode: "incremental",
		MaxConcurrency: 2, HumanDecisionRequired: true, GitLinkWrites: 0,
		Tasks: []ReviewAgentTask{
			{TaskID: "task-correctness", Role: "correctness", ReadOnly: true},
			{TaskID: "task-tests", Role: "tests", ReadOnly: true},
		},
	}
	now := time.Date(2026, 8, 1, 15, 0, 0, 0, time.UTC)
	run := RunReviewAgentPlan(context.Background(), plan, provider, time.Second, func() time.Time { return now })
	if run.Status != "ready_for_owner_review" || run.GitLinkWrites != 0 ||
		run.Synthesis.AssessmentsReceived != 2 || !run.Synthesis.HumanDecisionRequired {
		t.Fatalf("Agent run = %#v", run)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(invocations) != 2 {
		t.Fatalf("invocation count = %d", len(invocations))
	}
	for _, invocation := range invocations {
		if invocation.GitLinkWrites != 0 || !invocation.Task.ReadOnly || invocation.HeadSHA != "head-42" {
			t.Fatalf("unsafe invocation = %#v", invocation)
		}
	}
}

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
			SchemaVersion:     reviewAgentAssessmentSchema,
			RunID:             plan.RunID,
			TaskID:            "task-correctness",
			Role:              "correctness",
			HeadSHA:           plan.HeadSHA,
			Status:            "completed",
			AssessmentSummary: "Found a persistence risk in the reviewed patchset.",
			Coverage:          []string{"changed_files", "review_threads"},
			EvidenceChecked:   []string{"patchset", "existing_reviews"},
			CompletedAt:       "2026-07-31T08:10:00Z",
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
			Status:        "completed",
			CompletedAt:   "2026-07-31T08:10:00Z",
		},
	}
	result := SynthesizeReviewAssessments(plan, assessments)
	if result.Status != "incomplete" || len(result.Conflicts) == 0 ||
		result.Recommendation != "human_decision_required" ||
		!result.HumanDecisionRequired || result.GitLinkWrites != 0 {
		t.Fatalf("synthesis = %#v", result)
	}
}

func TestSynthesizeReviewAssessmentsRejectsEmptyCompletedAssessment(t *testing.T) {
	plan := ReviewAgentPlan{
		SchemaVersion: reviewAgentPlanSchema,
		RunID:         "run-empty",
		HeadSHA:       "head-empty",
		Tasks:         []ReviewAgentTask{{TaskID: "task-empty", Role: "correctness"}},
	}
	result := SynthesizeReviewAssessments(plan, []ReviewAgentAssessment{{
		SchemaVersion:     reviewAgentAssessmentSchema,
		RunID:             plan.RunID,
		TaskID:            "task-empty",
		Role:              "correctness",
		HeadSHA:           plan.HeadSHA,
		Status:            "completed",
		AssessmentSummary: "No issues reported.",
		Coverage:          []string{"changed_files"},
		EvidenceChecked:   []string{"patchset"},
		CompletedAt:       "2026-07-31T08:10:00Z",
	}})
	if result.Status != "incomplete" || result.AssessmentsReceived != 0 || len(result.Conflicts) != 1 {
		t.Fatalf("empty completed assessment was accepted: %#v", result)
	}
}

func TestSynthesizeReviewAssessmentsRejectsFailedOrIncompleteAssessment(t *testing.T) {
	plan := ReviewAgentPlan{
		SchemaVersion: reviewAgentPlanSchema,
		RunID:         "run-validation",
		HeadSHA:       "head-validation",
		Tasks: []ReviewAgentTask{
			{TaskID: "task-failed", Role: "correctness"},
			{TaskID: "task-invalid-finding", Role: "tests"},
		},
	}
	result := SynthesizeReviewAssessments(plan, []ReviewAgentAssessment{
		{
			SchemaVersion: reviewAgentAssessmentSchema,
			RunID:         plan.RunID,
			TaskID:        "task-failed",
			Role:          "correctness",
			HeadSHA:       plan.HeadSHA,
			Status:        "failed",
			CompletedAt:   "2026-07-31T08:10:00Z",
		},
		{
			SchemaVersion: reviewAgentAssessmentSchema,
			RunID:         plan.RunID,
			TaskID:        "task-invalid-finding",
			Role:          "tests",
			HeadSHA:       plan.HeadSHA,
			Status:        "completed",
			CompletedAt:   "2026-07-31T08:10:00Z",
			Findings: []ReviewAgentFinding{{
				Severity: "not-a-severity",
				Summary:  "missing evidence and confidence",
			}},
		},
	})
	if result.Status != "incomplete" || result.AssessmentsReceived != 0 ||
		len(result.Conflicts) != 2 || result.Recommendation != "human_decision_required" {
		t.Fatalf("invalid assessments were accepted: %#v", result)
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

func TestBuildReviewWarroomOverlaysP2CollaborationState(t *testing.T) {
	contexts := []ReviewContext{{
		Repository:       "owner/repo",
		PullRequest:      42,
		CurrentHeadSHA:   "head-42",
		CollectionStatus: "complete",
		WorkItem: ReviewWorkItem{
			PRKey:               "owner/repo#42",
			ReviewStage:         "human_reviewing",
			GitLinkState:        "open",
			RecommendedNextStep: "review current patchset",
			GeneratedAt:         "2026-07-31T08:00:00Z",
		},
	}}
	warroom := BuildReviewWarroomWithCollaboration(contexts, []collab.WorkItem{{
		SchemaVersion:       collab.WorkItemSchema,
		PRKey:               "owner/repo#42",
		Repository:          "owner/repo",
		PRNumber:            42,
		ReviewStage:         "human_reviewing",
		CollectionStatus:    "complete",
		AssignedTo:          "ou_reviewer",
		CollaborationStatus: "reviewing",
		DueAt:               "2026-08-02",
		NextStep:            "finish assigned Review",
	}}, time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC))
	if len(warroom.Items) != 1 {
		t.Fatalf("warroom items = %#v", warroom.Items)
	}
	item := warroom.Items[0]
	if item.AssignedTo != "ou_reviewer" || item.CollaborationStatus != "reviewing" ||
		item.DueAt != "2026-08-02" || item.NextStep != "finish assigned Review" {
		t.Fatalf("P2 collaboration overlay lost: %#v", item)
	}
}

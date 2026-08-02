package feishu

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func TestReviewGatewayResultCardGoldenStates(t *testing.T) {
	job, complete, item := reviewGatewayCardFixture()
	partial := complete
	partial.CollectionStatus = "partial"
	partial.Partial = true
	failed := complete
	failed.Status = "failed"
	failed.Error = "GitLink API request failed (HTTP 504)"
	failed.PullRequest = nil

	tests := []struct {
		name   string
		result ReviewGatewayExecutionResult
		item   *ReviewCollaborationItem
		golden string
	}{
		{name: "complete", result: complete, item: &item, golden: completeReviewGatewayCardGolden},
		{name: "partial", result: partial, item: &item, golden: partialReviewGatewayCardGolden},
		{name: "failed", result: failed, golden: failedReviewGatewayCardGolden},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			card := buildReviewGatewayResultCard(job, test.result, test.item)
			if len(card) == 0 {
				t.Fatal("card unexpectedly downgraded")
			}
			got := reviewGatewayCardSnapshot(card)
			gotGolden := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(got)))
			if gotGolden != test.golden {
				t.Fatalf("card golden mismatch: got %s, want %s\n--- snapshot ---\n%s", gotGolden, test.golden, got)
			}
		})
	}
}

func TestReviewGatewayResultCardStateAndPublicBoundaries(t *testing.T) {
	job, result, item := reviewGatewayCardFixture()
	for state, template := range map[string]string{"open": "orange", "merged": "green", "closed": "grey"} {
		t.Run(state, func(t *testing.T) {
			stateResult := result
			stateResult.GitLinkState = state
			card := buildReviewGatewayResultCard(job, stateResult, &item)
			header, _ := card["header"].(map[string]interface{})
			if header["template"] != template {
				t.Fatalf("template = %v, want %s", header["template"], template)
			}
		})
	}

	result.PublicRead = true
	publicJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok {
		t.Fatal("public card unexpectedly downgraded")
	}
	for _, forbidden := range []string{"Base / Doc / Task", "协作状态", "负责人", "截止时间", "ou_secret"} {
		if strings.Contains(publicJSON, forbidden) {
			t.Fatalf("public card leaked bound collaboration field %q: %s", forbidden, publicJSON)
		}
	}
	if !strings.Contains(publicJSON, "打开 GitLink PR") || !strings.Contains(publicJSON, "GitLink 写入：0") {
		t.Fatalf("public card lost navigation or write boundary: %s", publicJSON)
	}
}

func TestReviewGatewayResultCardHandlesUnreviewedPR(t *testing.T) {
	job, result, _ := reviewGatewayCardFixture()
	result.ReviewCount = 0
	result.ThreadCount = 0
	result.OpenThreadCount = 0
	result.Decision = "pending"
	result.PullRequest.Reviewers = nil
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(job, result, nil))
	if !ok || !strings.Contains(cardJSON, "**Review**\\n0") || strings.Contains(cardJSON, "Reviewer 摘要") {
		t.Fatalf("unreviewed PR card = %s, ok=%t", cardJSON, ok)
	}
}

func TestPopulateReviewGatewayResultBoundsPresentationData(t *testing.T) {
	contextInput := workflow.ReviewContext{
		Repository:       "owner/repo",
		PullRequest:      42,
		CurrentHeadSHA:   strings.Repeat("a", 40),
		CurrentVersionID: strings.Repeat("v", 80),
		CurrentPatchset: workflow.ReviewContextPatchset{
			FilesCount: 4, CommitsCount: 2, Additions: 31, Deletions: 9,
		},
		CollectionStatus: "complete",
		Summary: workflow.ReviewCollaborationSummary{
			Decision: "pending", TotalReviews: 12, TotalThreads: 8, OpenThreads: 3,
		},
		WorkItem: workflow.ReviewWorkItem{
			Title: strings.Repeat("长", 300), Author: "alice", BaseBranch: "main", HeadBranch: "feature",
			GitLinkState: "open", ReviewStage: "human_reviewing", RiskLevel: "high",
			Unknowns: []string{
				strings.Repeat("u", 300), "two", "three", "four", "five", "must-not-appear",
			},
			RecommendedNextStep: strings.Repeat("n", 400), SourceFingerprint: "fingerprint",
		},
	}
	for index := 0; index < 10; index++ {
		contextInput.ReviewerSummaries = append(contextInput.ReviewerSummaries, workflow.ReviewContextReviewerSummary{
			ReviewerKey: fmt.Sprintf("reviewer-%d", index), CurrentDecision: "commented",
		})
	}
	result := ReviewGatewayExecutionResult{}
	populateReviewGatewayContextResult(&result, contextInput)
	if result.PullRequest == nil || len([]rune(result.PullRequest.Title)) > 161 ||
		len(result.PullRequest.Unknowns) != 5 || len(result.PullRequest.Reviewers) != 8 {
		t.Fatalf("bounded view = %#v", result.PullRequest)
	}
	cardJSON, ok := safeReviewGatewayCardJSON(buildReviewGatewayResultCard(
		ReviewGatewayJob{Repository: "owner/repo", PRNumber: 42}, result, nil,
	))
	if !ok {
		t.Fatal("bounded presentation unexpectedly exceeded card budget")
	}
	for _, forbidden := range []string{strings.Repeat("a", 40), "must-not-appear", "reviewer-9"} {
		if strings.Contains(cardJSON, forbidden) {
			t.Fatalf("card contains unbounded or sensitive value %q", forbidden)
		}
	}
}

func TestReviewGatewayOversizedCardFallsBackToText(t *testing.T) {
	now := time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC)
	store := NewMemoryReviewGatewayJobStore()
	job := testReviewGatewayJob(now, "oversized-card")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker", Now: now, Limit: 1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion: reviewGatewayResultSchema, JobID: job.JobID, Status: "completed",
		Repository: job.Repository, PRNumber: job.PRNumber, ReviewStage: "triaged", Decision: "pending",
		ResultCard: Card{"oversized": strings.Repeat("x", reviewGatewayCardJSONLimit+1)},
	}
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(sender, store, &reviewGatewayJSONOutput{writer: io.Discard}, 1)
	dispatcher.now = func() time.Time { return now.Add(time.Second) }
	dispatcher.deliverPendingReplies(context.Background())
	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.inputs) != 1 || sender.inputs[0].MsgType != "text" ||
		!strings.Contains(sender.inputs[0].Text, "GitLink 写入：0") {
		t.Fatalf("oversized card did not safely fall back to text: %#v", sender.inputs)
	}
}

func reviewGatewayCardFixture() (ReviewGatewayJob, ReviewGatewayExecutionResult, ReviewCollaborationItem) {
	job := ReviewGatewayJob{Repository: "owner/repo", PRNumber: 42}
	result := ReviewGatewayExecutionResult{
		Status: "completed", Repository: "owner/repo", PRNumber: 42,
		CollectionStatus: "complete", HeadSHA: "1234567890abcdef1234567890abcdef12345678",
		ReviewStage: "human_reviewing", Decision: "pending", GitLinkState: "open",
		ReviewCount: 3, ThreadCount: 4, OpenThreadCount: 2,
		PullRequest: &ReviewGatewayPullRequestView{
			Title: "Add safe review gateway", Author: "alice", BaseBranch: "main", HeadBranch: "feature/card",
			GitLinkURL: "https://www.gitlink.org.cn/owner/repo/pulls/42", PatchsetID: "7",
			FilesCount: 5, CommitsCount: 2, Additions: 120, Deletions: 18, RiskLevel: "high",
			RecommendedNextStep: "由 Owner 团队完成高风险文件复核。",
			Unknowns:            []string{"CI 状态未返回"},
			Reviewers: []ReviewGatewayReviewerView{
				{Reviewer: "bob", Decision: "approved"},
				{Reviewer: "carol", Decision: "commented"},
			},
		},
	}
	item := ReviewCollaborationItem{
		Repository: "owner/repo", PRNumber: 42, ReviewStage: "human_reviewing", Decision: "pending",
		CollaborationStatus: "reviewing", AssignedTo: "ou_secret", DueAt: "2026-08-05",
	}
	return job, result, item
}

func reviewGatewayCardSnapshot(card Card) string {
	header, _ := card["header"].(map[string]interface{})
	title, _ := header["title"].(map[string]interface{})
	lines := []string{
		fmt.Sprintf("template=%v", header["template"]),
		fmt.Sprintf("title=%v", title["content"]),
	}
	elements, _ := card["elements"].([]interface{})
	for index, raw := range elements {
		element, _ := raw.(map[string]interface{})
		lines = append(lines, fmt.Sprintf("element[%d]=%v", index, element["tag"]))
		if text, ok := element["text"].(map[string]interface{}); ok {
			lines = append(lines, fmt.Sprintf("text=%q", text["content"]))
		}
		if rawFields, ok := element["fields"].([]interface{}); ok {
			for _, rawField := range rawFields {
				field, _ := rawField.(map[string]interface{})
				text, _ := field["text"].(map[string]interface{})
				lines = append(lines, fmt.Sprintf("field=%q", text["content"]))
			}
		}
		if rawNote, ok := element["elements"].([]interface{}); ok {
			for _, rawPart := range rawNote {
				part, _ := rawPart.(map[string]interface{})
				lines = append(lines, fmt.Sprintf("note=%q", part["content"]))
			}
		}
		if rawActions, ok := element["actions"].([]interface{}); ok {
			for _, rawAction := range rawActions {
				action, _ := rawAction.(map[string]interface{})
				text, _ := action["text"].(map[string]interface{})
				lines = append(lines, fmt.Sprintf("button=%q url=%q", text["content"], action["url"]))
			}
		}
	}
	return strings.Join(lines, "\n")
}

const completeReviewGatewayCardGolden = "sha256:54db14e937f887f00b54ce6aeab119b467ce275f1243ba056e2b88537c4fe4c8"
const partialReviewGatewayCardGolden = "sha256:89f709dceca7a7f70f5dd288a6bc2069d1c67944f4dfad3e5dac0a4c9098afe8"
const failedReviewGatewayCardGolden = "sha256:4ccc58ff3193eea4d7540b1ecef75f67c63c616a57cccabb8c9b14555206e1e4"

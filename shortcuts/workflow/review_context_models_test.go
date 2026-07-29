package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEvaluateReviewFreshness(t *testing.T) {
	tests := []struct {
		name     string
		commitID string
		headSHA  string
		want     string
	}{
		{name: "exact", commitID: "abc123def456", headSHA: "abc123def456", want: reviewFreshnessCurrent},
		{name: "abbreviated review", commitID: "abc123d", headSHA: "abc123def456", want: reviewFreshnessCurrent},
		{name: "abbreviated head", commitID: "abc123def456", headSHA: "abc123d", want: reviewFreshnessCurrent},
		{name: "outdated", commitID: "abc123def456", headSHA: "def456abc123", want: reviewFreshnessOutdated},
		{name: "missing review commit", commitID: "", headSHA: "abc123def456", want: reviewFreshnessUnknown},
		{name: "missing head", commitID: "abc123def456", headSHA: "", want: reviewFreshnessUnknown},
		{name: "unsafe short prefix", commitID: "abc", headSHA: "abcdef123", want: reviewFreshnessOutdated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := evaluateReviewFreshness(test.commitID, test.headSHA); got != test.want {
				t.Fatalf("evaluateReviewFreshness(%q, %q) = %q, want %q", test.commitID, test.headSHA, got, test.want)
			}
		})
	}
}

func TestSummarizeReviewCollaborationUsesOnlyCurrentEvidence(t *testing.T) {
	needResponse := true
	reviews := []ReviewContextReview{
		{ID: "1", Status: "approved", CommitID: "head1234", Freshness: reviewFreshnessCurrent},
		{ID: "2", Status: "rejected", CommitID: "old12345", Freshness: reviewFreshnessOutdated},
		{ID: "3", Status: "common", Freshness: reviewFreshnessUnknown},
	}
	threads := []ReviewContextThread{
		{ID: "10", State: "opened", NeedRespond: &needResponse, Freshness: reviewFreshnessCurrent},
		{ID: "11", State: "resolved", Freshness: reviewFreshnessOutdated},
		{ID: "12", ParentID: "missing", State: "opened", Freshness: reviewFreshnessUnknown, UnknownParent: true},
	}

	got := summarizeReviewCollaboration("open", reviews, threads, true, true)
	if got.GitLinkReviewStatus != "approved" {
		t.Fatalf("GitLinkReviewStatus = %q, want approved", got.GitLinkReviewStatus)
	}
	if got.Decision != "changes_pending" {
		t.Fatalf("Decision = %q, want changes_pending", got.Decision)
	}
	if got.CurrentReviews != 1 || got.OutdatedReviews != 1 || got.UnknownReviews != 1 {
		t.Fatalf("review freshness counts = %+v", got)
	}
	if got.PendingResponse != 1 || got.UnknownParentReplies != 1 {
		t.Fatalf("thread counts = %+v", got)
	}
	if got.CurrentThreads != 1 || got.OutdatedThreads != 1 || got.UnknownThreads != 1 {
		t.Fatalf("thread freshness counts = %+v", got)
	}
}

func TestSummarizeReviewCollaborationCurrentRejectionBlocks(t *testing.T) {
	got := summarizeReviewCollaboration("open", []ReviewContextReview{
		{Status: "approved", Freshness: reviewFreshnessCurrent},
		{Status: "rejected", Freshness: reviewFreshnessCurrent},
	}, nil, true, true)
	if got.GitLinkReviewStatus != "rejected" || got.Decision != "blocked" {
		t.Fatalf("summary = %+v, want rejected/blocked", got)
	}
}

func TestFinalizeReviewContextFingerprintIsOrderIndependent(t *testing.T) {
	now := time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC)
	base := ReviewContext{
		Repository:  "owner/repo",
		PullRequest: 42,
		Source:      "fixture",
		Sections:    []string{"pr", "reviews", "threads"},
		PR: map[string]interface{}{
			"number":          42,
			"title":           "feat: review context",
			"state":           "open",
			"head_commit_sha": "abc123def456",
		},
		Reviews: []map[string]interface{}{
			{"id": 2, "status": "common", "commit_id": "abc123def456", "user": map[string]interface{}{"login": "bob"}},
			{"id": 1, "status": "approved", "commit_id": "abc123def456", "user": map[string]interface{}{"login": "alice"}},
		},
		Threads: []ReviewContextThread{
			{ID: "11", State: "resolved", CommitID: "abc123def456"},
			{ID: "10", State: "opened", CommitID: "abc123def456"},
		},
	}
	reordered := base
	reordered.Reviews = []map[string]interface{}{base.Reviews[1], base.Reviews[0]}
	reordered.Threads = []ReviewContextThread{base.Threads[1], base.Threads[0]}

	finalizeReviewContext(&base, now)
	finalizeReviewContext(&reordered, now.Add(time.Hour))

	if base.WorkItem.SourceFingerprint != reordered.WorkItem.SourceFingerprint {
		t.Fatalf("fingerprints differ by input order:\n%s\n%s", base.WorkItem.SourceFingerprint, reordered.WorkItem.SourceFingerprint)
	}
	if !strings.HasPrefix(base.WorkItem.SourceFingerprint, "sha256:") {
		t.Fatalf("fingerprint = %q", base.WorkItem.SourceFingerprint)
	}
	if base.WorkItem.GeneratedAt == reordered.WorkItem.GeneratedAt {
		t.Fatal("generated_at should remain observation time, not fingerprint input")
	}
}

func TestFinalizeReviewContextKeepsUnknownsConservative(t *testing.T) {
	context := ReviewContext{
		Repository:  "owner/repo",
		PullRequest: 9,
		Sections:    []string{"pr", "reviews"},
		PR:          map[string]interface{}{"number": 9, "state": "open"},
		Reviews:     []map[string]interface{}{{"id": 1, "status": "approved"}},
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	if context.Summary.ReviewFreshness != reviewFreshnessUnknown || context.Summary.Decision != "pending" {
		t.Fatalf("summary = %+v, want unknown/pending", context.Summary)
	}
	if context.WorkItem.Mergeability != "unknown" || context.WorkItem.ReviewStage == "ready_for_decision" {
		t.Fatalf("work item must remain conservative: %+v", context.WorkItem)
	}
	if !containsString(context.WorkItem.Unknowns, "current_head_sha_not_returned") {
		t.Fatalf("unknowns = %v, want missing head", context.WorkItem.Unknowns)
	}
	if !containsString(context.WorkItem.Unknowns, "review_threads_not_loaded") {
		t.Fatalf("unknowns = %v, want missing threads", context.WorkItem.Unknowns)
	}
	if !containsString(context.WorkItem.Unknowns, "changed_files_not_loaded") {
		t.Fatalf("unknowns = %v, want missing changed files", context.WorkItem.Unknowns)
	}
}

func TestFinalizeReviewContextMarksConfiguredLimitAsSampled(t *testing.T) {
	context := ReviewContext{
		Repository:  "owner/repo",
		PullRequest: 10,
		Sections:    []string{"pr", "reviews", "threads"},
		PR: map[string]interface{}{
			"number":          10,
			"state":           "open",
			"head_commit_sha": "abc123def456",
		},
		Reviews:     []map[string]interface{}{},
		Threads:     []ReviewContextThread{{ID: "1", State: "resolved", CommitID: "abc123def456"}},
		threadLimit: 1,
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	if !context.WorkItem.SourceScope.Sampled || context.WorkItem.SourceScope.Complete {
		t.Fatalf("source scope = %+v, want sampled and incomplete", context.WorkItem.SourceScope)
	}
}

func TestNormalizeReviewContextThreadsMarksOrphanReplies(t *testing.T) {
	threads := normalizeReviewContextThreads([]map[string]interface{}{
		{"id": 1, "state": "opened", "commit_id": "abc123def456"},
		{"id": 2, "parent_id": 1, "state": "opened", "commit_id": "abc123def456"},
		{"id": 3, "parent_id": 99, "state": "opened", "commit_id": "abc123def456"},
	}, "abc123def456")

	if len(threads) != 3 {
		t.Fatalf("threads = %d, want 3", len(threads))
	}
	byID := map[string]ReviewContextThread{}
	for _, thread := range threads {
		byID[thread.ID] = thread
	}
	if byID["2"].UnknownParent {
		t.Fatal("reply to known parent marked orphan")
	}
	if !byID["3"].UnknownParent {
		t.Fatal("reply to missing parent not marked orphan")
	}
}

func TestReviewContextP1FixtureMatchesMarkdownGolden(t *testing.T) {
	fixturePath := filepath.Join("testdata", "review_context_p1_fixture.json")
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var context ReviewContext
	if err := json.Unmarshal(data, &context); err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	got, err := RenderReviewContext(context, "markdown")
	if err != nil {
		t.Fatalf("RenderReviewContext: %v", err)
	}
	goldenPath := filepath.Join("testdata", "review_context_p1.golden.md")
	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	normalize := func(value string) string {
		return strings.ReplaceAll(value, "\r\n", "\n")
	}
	if normalize(got) != normalize(string(want)) {
		t.Fatalf("markdown output does not match golden\n--- got ---\n%s\n--- want ---\n%s", got, string(want))
	}
}

func TestReadReviewContextInputSupportsOfflineFixture(t *testing.T) {
	context, err := readReviewContextInput(filepath.Join("testdata", "review_context_p1_fixture.json"))
	if err != nil {
		t.Fatalf("readReviewContextInput: %v", err)
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	if context.Repository != "owner/repo" || context.PullRequest != 42 {
		t.Fatalf("context target = %s#%d", context.Repository, context.PullRequest)
	}
	if context.SchemaVersion != reviewContextSchemaVersion || context.WorkItem.SourceFingerprint == "" {
		t.Fatalf("offline context was not finalized: %+v", context)
	}
}

func TestFinalizeReviewContextPreservesTypedOfflineRecords(t *testing.T) {
	context := ReviewContext{
		Repository:     "owner/repo",
		PullRequest:    42,
		Source:         "local-json",
		Sections:       []string{"reviews", "threads"},
		CurrentHeadSHA: "abc123def456",
		ReviewRecords: []ReviewContextReview{{
			ID: "1", Status: "approved", CommitID: "abc123def456", SourceType: "formal_review",
		}},
		Threads: []ReviewContextThread{{
			ID: "2", State: "resolved", CommitID: "abc123def456",
		}},
		WorkItem: ReviewWorkItem{GitLinkState: "open"},
	}
	finalizeReviewContext(&context, time.Date(2026, 7, 29, 12, 0, 0, 0, time.UTC))

	if len(context.ReviewRecords) != 1 || context.ReviewRecords[0].Freshness != reviewFreshnessCurrent {
		t.Fatalf("typed review records were not preserved: %+v", context.ReviewRecords)
	}
	if context.CurrentHeadSHA != "abc123def456" || context.Summary.Decision != "approved" {
		t.Fatalf("offline context summary = %+v", context)
	}
}

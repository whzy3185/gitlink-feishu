package feishu

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func openReviewPresentationTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "presentation.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	return store
}

func TestCompleteResultPersistsFullPresentation(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-complete")
	want := fullReviewGatewayResultFixture()
	got, err := store.SaveReviewPRPresentation(context.Background(), job, want, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("SaveReviewPRPresentation: %v", err)
	}
	if !got.Complete() || !got.Matches(job) || got.Title != want.PullRequest.Title ||
		got.HeadSHA != want.HeadSHA || got.FilesCount != want.PullRequest.FilesCount ||
		got.OpenThreadCount != want.OpenThreadCount || got.ContentFingerprint == "" {
		t.Fatalf("incomplete persisted presentation: %#v", got)
	}
}

func TestPartialResultDoesNotReplaceCompletePresentation(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-partial")
	complete := fullReviewGatewayResultFixture()
	want, err := store.SaveReviewPRPresentation(context.Background(), job, complete, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("save complete: %v", err)
	}
	partial := fullReviewGatewayResultFixture()
	partial.CollectionStatus = "partial"
	partial.Partial = true
	partial.HeadSHA = "partial-head"
	partial.PullRequest.Title = "partial replacement"
	partial.PullRequest.Unknowns = []string{"partial uncertainty"}
	got, err := store.SaveReviewPRPresentation(context.Background(), job, partial, reviewProductInvariantTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("save partial: %v", err)
	}
	if !got.Complete() || got.ContentFingerprint != want.ContentFingerprint || got.UpdatedAt != want.UpdatedAt ||
		got.Title != want.Title || got.HeadSHA != want.HeadSHA || !reflect.DeepEqual(got.Unknowns, want.Unknowns) {
		t.Fatalf("partial result replaced complete presentation: before=%#v after=%#v", want, got)
	}
}

func TestFailedResultPreservesCompletePresentation(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-failed")
	want, err := store.SaveReviewPRPresentation(context.Background(), job, fullReviewGatewayResultFixture(), reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("save complete: %v", err)
	}
	failed := fullReviewGatewayResultFixture()
	failed.Status = "failed"
	failed.Error = "temporary upstream error"
	failed.PullRequest.Title = "must not replace"
	got, err := store.SaveReviewPRPresentation(context.Background(), job, failed, reviewProductInvariantTime.Add(time.Minute))
	if err != nil {
		t.Fatalf("save failed result: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("failed result changed complete presentation: before=%#v after=%#v", want, got)
	}
}

func TestPresentationIsIsolatedByInstallation(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	jobA := testReviewGatewayJob(reviewProductInvariantTime, "presentation-install-a")
	jobA.InstallationID = "installation-a"
	jobB := jobA
	jobB.JobID = "job-presentation-install-b"
	jobB.InstallationID = "installation-b"
	resultA := fullReviewGatewayResultFixture()
	resultB := fullReviewGatewayResultFixture()
	resultA.PullRequest.Title = "installation A"
	resultB.PullRequest.Title = "installation B"
	if _, err := store.SaveReviewPRPresentation(context.Background(), jobA, resultA, reviewProductInvariantTime); err != nil {
		t.Fatalf("save A: %v", err)
	}
	if _, err := store.SaveReviewPRPresentation(context.Background(), jobB, resultB, reviewProductInvariantTime); err != nil {
		t.Fatalf("save B: %v", err)
	}
	gotA, _ := store.GetReviewPRPresentation(context.Background(), jobA)
	gotB, _ := store.GetReviewPRPresentation(context.Background(), jobB)
	if gotA.PresentationKey == gotB.PresentationKey || gotA.Title != "installation A" || gotB.Title != "installation B" {
		t.Fatalf("installation presentations leaked: A=%#v B=%#v", gotA, gotB)
	}
}

func TestPresentationIsSharedAcrossChats(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	jobA := testReviewGatewayJob(reviewProductInvariantTime, "presentation-chat-a")
	jobA.ChatID = "oc_a"
	jobB := jobA
	jobB.ChatID = "oc_b"
	if _, err := store.SaveReviewPRPresentation(context.Background(), jobA, fullReviewGatewayResultFixture(), reviewProductInvariantTime); err != nil {
		t.Fatalf("save chat A: %v", err)
	}
	gotA, _ := store.GetReviewPRPresentation(context.Background(), jobA)
	gotB, _ := store.GetReviewPRPresentation(context.Background(), jobB)
	if !reflect.DeepEqual(gotA, gotB) || gotA.PresentationKey == "" {
		t.Fatalf("presentation was not shared across chats: A=%#v B=%#v", gotA, gotB)
	}
}

func TestPresentationContentFingerprintIsDeterministic(t *testing.T) {
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-fingerprint")
	left := fullReviewGatewayResultFixture()
	left.PullRequest.Reviewers = append(left.PullRequest.Reviewers,
		ReviewGatewayReviewerView{Reviewer: "alice", Decision: "approved"})
	left.PullRequest.Unknowns = []string{"z", "a"}
	right := fullReviewGatewayResultFixture()
	right.PullRequest.Reviewers = []ReviewGatewayReviewerView{
		{Reviewer: "alice", Decision: "approved"},
		{Reviewer: "bob", Decision: "changes_pending"},
	}
	right.PullRequest.Unknowns = []string{"a", "z"}
	leftPresentation, err := reviewPRPresentationFromResult(job, left, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("map left presentation: %v", err)
	}
	rightPresentation, err := reviewPRPresentationFromResult(job, right, reviewProductInvariantTime.Add(time.Hour))
	if err != nil {
		t.Fatalf("map right presentation: %v", err)
	}
	if leftPresentation.ContentFingerprint != rightPresentation.ContentFingerprint {
		t.Fatalf("equivalent presentation fingerprints differ: %s != %s", leftPresentation.ContentFingerprint, rightPresentation.ContentFingerprint)
	}
}

func TestPresentationBoundsReviewersAndUnknowns(t *testing.T) {
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-bounds")
	result := fullReviewGatewayResultFixture()
	result.PullRequest.Title = strings.Repeat("题", 250)
	result.PullRequest.Author = strings.Repeat("人", 150)
	result.PullRequest.BaseBranch = strings.Repeat("b", 250)
	result.PullRequest.HeadBranch = strings.Repeat("h", 250)
	result.PullRequest.RecommendedNextStep = strings.Repeat("n", 600)
	result.PullRequest.GitLinkURL = "https://www.gitlink.org.cn/" + strings.Repeat("u", 600)
	result.PullRequest.Reviewers = nil
	result.PullRequest.Unknowns = nil
	for index := 0; index < 25; index++ {
		result.PullRequest.Reviewers = append(result.PullRequest.Reviewers, ReviewGatewayReviewerView{
			Reviewer: fmt.Sprintf("%02d-%s", index, strings.Repeat("r", 120)), Decision: "pending",
		})
		result.PullRequest.Unknowns = append(result.PullRequest.Unknowns, fmt.Sprintf("%02d-%s", index, strings.Repeat("u", 350)))
	}
	presentation, err := reviewPRPresentationFromResult(job, result, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("map bounded presentation: %v", err)
	}
	if len([]rune(presentation.Title)) > 200 || len([]rune(presentation.Author)) > 100 ||
		len([]rune(presentation.BaseBranch)) > 200 || len([]rune(presentation.HeadBranch)) > 200 ||
		len([]rune(presentation.RecommendedNextStep)) > 500 || len([]rune(presentation.GitLinkURL)) > 500 ||
		len(presentation.Reviewers) != 25 || len(presentation.Unknowns) != 20 {
		t.Fatalf("presentation bounds not enforced: %#v", presentation)
	}
	for _, unknown := range presentation.Unknowns {
		if len([]rune(unknown)) > 300 {
			t.Fatalf("unknown exceeds 300 characters: %d", len([]rune(unknown)))
		}
	}
}

func TestPartialPresentationCannotAuthorizeCollaborationAction(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "partial-not-authoritative")
	partial := fullReviewGatewayResultFixture()
	partial.CollectionStatus = "partial"
	partial.Partial = true
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, partial, reviewProductInvariantTime); err != nil {
		t.Fatalf("save partial presentation: %v", err)
	}
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute)); !errors.Is(err, ErrCompleteReviewPRPresentationRequired) {
		t.Fatalf("partial presentation action error = %v", err)
	}
}

func TestPresentationTruncatesLongText(t *testing.T) {
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-truncation")
	result := fullReviewGatewayResultFixture()
	result.HeadSHA = strings.Repeat("h", 80)
	result.GitLinkState = strings.Repeat("s", 40)
	result.ReviewStage = strings.Repeat("r", 80)
	result.Decision = strings.Repeat("d", 80)
	result.PullRequest.PatchsetID = strings.Repeat("p", 120)
	result.PullRequest.RiskLevel = strings.Repeat("k", 40)
	presentation, err := reviewPRPresentationFromResult(job, result, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("reviewPRPresentationFromResult: %v", err)
	}
	limits := map[string]struct {
		value string
		limit int
	}{
		"head": {presentation.HeadSHA, 64}, "state": {presentation.GitLinkState, 32},
		"stage": {presentation.ReviewStage, 64}, "decision": {presentation.Decision, 64},
		"patchset": {presentation.PatchsetID, 100}, "risk": {presentation.RiskLevel, 32},
	}
	for name, item := range limits {
		if len([]rune(item.value)) != item.limit {
			t.Fatalf("%s length = %d, want %d", name, len([]rune(item.value)), item.limit)
		}
	}
}

func TestPresentationRejectsInvalidJSON(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-invalid-json")
	presentation, err := store.SaveReviewPRPresentation(context.Background(), job, fullReviewGatewayResultFixture(), reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("save presentation: %v", err)
	}
	if _, err := store.db.Exec(`UPDATE review_pr_presentations SET reviewers_json='{' WHERE presentation_key=?`, presentation.PresentationKey); err != nil {
		t.Fatalf("corrupt reviewers_json: %v", err)
	}
	if _, err := store.GetReviewPRPresentation(context.Background(), job); err == nil || !strings.Contains(err.Error(), "decode") {
		t.Fatalf("invalid JSON did not fail explicitly: %v", err)
	}
}

func TestPresentationDoesNotPersistSecretsOrRawPayload(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-secret-boundary")
	result := fullReviewGatewayResultFixture()
	result.PullRequest.Unknowns = []string{"authorization: bearer super-secret-value"}
	presentation, err := store.SaveReviewPRPresentation(context.Background(), job, result, reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("save presentation: %v", err)
	}
	encoded := fmt.Sprintf("%#v", presentation)
	if strings.Contains(encoded, "super-secret-value") || strings.Contains(encoded, job.ChatID) || strings.Contains(encoded, job.RequestedBy) {
		t.Fatalf("presentation persisted a secret or chat/user identifier: %s", encoded)
	}
	for _, forbidden := range []string{"payload_json", "chat_id", "message_id", "token", "cookie", "authorization"} {
		var count int
		if err := store.db.QueryRow(
			`SELECT COUNT(*) FROM pragma_table_info('review_pr_presentations') WHERE lower(name)=?`, forbidden,
		).Scan(&count); err != nil || count != 0 {
			t.Fatalf("forbidden presentation column %q exists: count=%d err=%v", forbidden, count, err)
		}
	}
}

func TestStaleCompletedJobDoesNotOverwriteNewPresentation(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-stale")
	newer := fullReviewGatewayResultFixture()
	newer.CompletedAt = reviewProductInvariantTime.Add(2 * time.Hour).Format(time.RFC3339)
	newer.HeadSHA = "newer-head"
	newer.SourceFingerprint = "newer-source"
	newer.PullRequest.Title = "newer presentation"
	want, err := store.SaveReviewPRPresentation(context.Background(), job, newer, reviewProductInvariantTime.Add(2*time.Hour))
	if err != nil {
		t.Fatalf("save newer presentation: %v", err)
	}
	stale := fullReviewGatewayResultFixture()
	stale.CompletedAt = reviewProductInvariantTime.Add(time.Hour).Format(time.RFC3339)
	stale.HeadSHA = "stale-head"
	stale.SourceFingerprint = "stale-source"
	stale.PullRequest.Title = "stale presentation"
	if _, err := store.SaveReviewPRPresentation(context.Background(), job, stale, reviewProductInvariantTime.Add(3*time.Hour)); !errors.Is(err, ErrStaleReviewPRPresentation) {
		t.Fatalf("stale presentation error = %v", err)
	}
	got, err := store.GetReviewPRPresentation(context.Background(), job)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("stale result changed presentation: got=%#v want=%#v err=%v", got, want, err)
	}
}

func TestPresentationSurvivesSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presentation-restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open first store: %v", err)
	}
	job := testReviewGatewayJob(reviewProductInvariantTime, "presentation-restart")
	want, err := store.SaveReviewPRPresentation(context.Background(), job, fullReviewGatewayResultFixture(), reviewProductInvariantTime)
	if err != nil {
		t.Fatalf("save presentation: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close first store: %v", err)
	}
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer store.Close()
	got, err := store.GetReviewPRPresentation(context.Background(), job)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("presentation did not survive restart: got=%#v want=%#v err=%v", got, want, err)
	}
}

func TestPresentationMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "presentation-migration.db")
	for pass := 0; pass < 2; pass++ {
		store, err := OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("open pass %d: %v", pass+1, err)
		}
		_ = store.Close()
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open migration database: %v", err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE name='review_pr_presentations_v1'`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("presentation migration count=%d err=%v", count, err)
	}
}

func TestMigrationRollbackLeavesOldDatabaseUsable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migration-rollback.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE schema_migrations (version INTEGER PRIMARY KEY, name TEXT NOT NULL UNIQUE, applied_at TEXT NOT NULL); CREATE TABLE legacy_keep(value TEXT); INSERT INTO legacy_keep(value) VALUES('kept')`); err != nil {
		t.Fatalf("seed legacy database: %v", err)
	}
	err = applyReviewGatewaySchemaMigrations(db, []reviewGatewaySchemaMigration{{
		Version: 99, Name: "failing_migration", Statements: []string{
			`INSERT INTO legacy_keep(value) VALUES('rolled-back')`,
			`INSERT INTO table_that_does_not_exist(value) VALUES('fail')`,
		},
	}})
	if err == nil {
		t.Fatal("failing migration unexpectedly succeeded")
	}
	var kept, migrationCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM legacy_keep WHERE value='kept'`).Scan(&kept); err != nil {
		t.Fatalf("read legacy row: %v", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=99`).Scan(&migrationCount); err != nil {
		t.Fatalf("read migration record: %v", err)
	}
	var rolledBack int
	_ = db.QueryRow(`SELECT COUNT(*) FROM legacy_keep WHERE value='rolled-back'`).Scan(&rolledBack)
	if kept != 1 || migrationCount != 0 || rolledBack != 0 {
		t.Fatalf("migration rollback failed: kept=%d migration=%d rolled_back=%d", kept, migrationCount, rolledBack)
	}
}

func TestExistingDatabaseUpgradesWithoutDeletingLegacyTables(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-upgrade.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE legacy_business_data(id TEXT PRIMARY KEY, remote_id TEXT); INSERT INTO legacy_business_data VALUES('one', 'remote-keep')`); err != nil {
		t.Fatalf("seed legacy table: %v", err)
	}
	_ = db.Close()
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("upgrade legacy database: %v", err)
	}
	defer store.Close()
	var remoteID string
	if err := store.db.QueryRow(`SELECT remote_id FROM legacy_business_data WHERE id='one'`).Scan(&remoteID); err != nil || remoteID != "remote-keep" {
		t.Fatalf("legacy table was changed: remote_id=%q err=%v", remoteID, err)
	}
}

func TestCollaborationFailureLeavesStateUnchanged(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime); err != nil {
		t.Fatalf("initial claim: %v", err)
	}
	before, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if err != nil || len(before) != 1 {
		t.Fatalf("read state before failure: %#v, %v", before, err)
	}
	conflict := job
	conflict.JobID = "job-conflicting-claim"
	conflict.RequestedBy = "ou_other"
	if _, err := store.ApplyCollaborationAction(context.Background(), conflict, reviewProductInvariantTime.Add(time.Minute)); err == nil {
		t.Fatal("conflicting claim unexpectedly succeeded")
	}
	after, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("failed collaboration changed state: before=%#v after=%#v err=%v", before, after, err)
	}
}

func TestCollaborationWithoutCompletePresentationLeavesStateUnchanged(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "collaboration-no-presentation")
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime); !errors.Is(err, ErrCompleteReviewPRPresentationRequired) {
		t.Fatalf("missing presentation error = %v", err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_collaboration_states`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("missing presentation changed collaboration state: count=%d err=%v", count, err)
	}
}

func TestPartialPresentationLeavesAssigneeUnchanged(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "partial-assignee")
	partial := fullReviewGatewayResultFixture()
	partial.CollectionStatus = "partial"
	partial.Partial = true
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, partial, reviewProductInvariantTime); err != nil {
		t.Fatalf("save partial: %v", err)
	}
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute)); !errors.Is(err, ErrCompleteReviewPRPresentationRequired) {
		t.Fatalf("partial action error = %v", err)
	}
	items, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if err != nil || len(items) != 1 || items[0].AssignedTo != "" || items[0].CollaborationStatus != "unassigned" {
		t.Fatalf("partial action changed assignee: items=%#v err=%v", items, err)
	}
}

func TestPresentationInstallationMismatchIsRejected(t *testing.T) {
	testPresentationScopeMismatchIsRejected(t, "installation_id", "other-installation")
}

func TestPresentationRepositoryMismatchIsRejected(t *testing.T) {
	testPresentationScopeMismatchIsRejected(t, "repository", "other/repo")
}

func TestPresentationPRNumberMismatchIsRejected(t *testing.T) {
	testPresentationScopeMismatchIsRejected(t, "pr_number", 99)
}

func testPresentationScopeMismatchIsRejected(t *testing.T, column string, value interface{}) {
	t.Helper()
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	if _, err := store.db.Exec(
		fmt.Sprintf("UPDATE review_pr_presentations SET %s=? WHERE presentation_key=?", column),
		value, reviewPRPresentationKey(job.InstallationID, job.Repository, job.PRNumber),
	); err != nil {
		t.Fatalf("corrupt presentation scope: %v", err)
	}
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute)); !errors.Is(err, ErrCompleteReviewPRPresentationRequired) {
		t.Fatalf("scope mismatch error = %v", err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_collaboration_states WHERE assigned_to<>''`).Scan(&count); err != nil || count != 0 {
		t.Fatalf("scope mismatch changed assignee: count=%d err=%v", count, err)
	}
}

func TestConcurrentClaimAllowsOnlyOneAssignee(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	start := make(chan struct{})
	type claimResult struct {
		item ReviewCollaborationItem
		err  error
	}
	results := make(chan claimResult, 2)
	for _, userID := range []string{"ou_alice", "ou_bob"} {
		claimJob := job
		claimJob.JobID = "job-concurrent-" + userID
		claimJob.Action = "claim_review"
		claimJob.RequestedBy = userID
		go func() {
			<-start
			item, err := store.ApplyCollaborationAction(context.Background(), claimJob, reviewProductInvariantTime.Add(time.Minute))
			results <- claimResult{item: item, err: err}
		}()
	}
	close(start)
	successes := 0
	failures := 0
	for index := 0; index < 2; index++ {
		result := <-results
		if result.err == nil {
			successes++
		} else {
			failures++
		}
	}
	items, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if successes != 1 || failures != 1 || err != nil || len(items) != 1 || items[0].AssignedTo == "" {
		t.Fatalf("concurrent claim invariant failed: success=%d failure=%d items=%#v err=%v", successes, failures, items, err)
	}
}

func TestArchivedPRRejectsClaim(t *testing.T) {
	store := openReviewPresentationTestStore(t)
	defer store.Close()
	job := testReviewGatewayJob(reviewProductInvariantTime, "archived-claim")
	result := fullReviewGatewayResultFixture()
	result.GitLinkState = "merged"
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, result, reviewProductInvariantTime); err != nil {
		t.Fatalf("save archived presentation: %v", err)
	}
	job.Action = "claim_review"
	if _, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute)); err == nil || !strings.Contains(err.Error(), "archived") {
		t.Fatalf("archived claim error = %v", err)
	}
}

func TestCollaborationActionPreservesGitLinkFacts(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	want, err := store.GetReviewPRPresentation(context.Background(), job)
	if err != nil {
		t.Fatalf("read presentation before action: %v", err)
	}
	result := executeProductInvariantAction(t, store, job, "claim_review", "", reviewProductInvariantTime.Add(time.Minute))
	got, err := store.GetReviewPRPresentation(context.Background(), job)
	if err != nil {
		t.Fatalf("read presentation after action: %v", err)
	}
	if !reflect.DeepEqual(got, want) || result.HeadSHA != want.HeadSHA || result.SourceFingerprint != want.SourceFingerprint || result.PullRequest == nil || result.PullRequest.Title != want.Title {
		t.Fatalf("collaboration action modified GitLink facts: before=%#v after=%#v result=%#v", want, got, result)
	}
}

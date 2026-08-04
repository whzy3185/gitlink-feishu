package feishu

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var reviewMigrationTestTime = time.Date(2026, 8, 4, 10, 0, 0, 0, time.UTC)

type fixtureLegacyReviewResourceVerifier struct{}

func (fixtureLegacyReviewResourceVerifier) Verify(
	_ context.Context,
	migration ReviewResourceMigration,
	input LegacyReviewResourceVerificationInput,
) error {
	if !input.Confirmed || input.Method != "fixture_verified" || input.MigrationID != migration.MigrationID {
		return errors.New("fixture verification rejected")
	}
	if input.ExpectedInstallation != migration.InstallationID || input.ExpectedScope != migration.TargetScope {
		return errors.New("fixture scope mismatch")
	}
	if migration.TargetScope == ReviewResourceScopeChat && reviewResourceIdentifierHash(input.ExpectedChatID) != migration.ChatIDHash {
		return errors.New("fixture chat mismatch")
	}
	return nil
}

func newReviewMigrationTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "migration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func seedReviewMigrationPlan(
	t *testing.T,
	store *SQLiteReviewGatewayStore,
	chatID string,
	installations []string,
	resource string,
	policy *ReviewResourceScopePolicy,
) ReviewResourceMigration {
	t.Helper()
	if policy != nil {
		if _, err := store.SetReviewResourceScopePolicy(context.Background(), *policy, reviewMigrationTestTime); err != nil {
			t.Fatalf("set policy: %v", err)
		}
	}
	insertLegacyProductInvariantItem(t, store, chatID, installations, []string{resource})
	if err := store.ScanLegacyReviewResourceMigrations(context.Background(), reviewMigrationTestTime); err != nil {
		t.Fatalf("scan migrations: %v", err)
	}
	plans, err := store.ListReviewResourceMigrations(context.Background(), "", resource, "")
	if err != nil || len(plans) != 1 {
		t.Fatalf("plans=%#v err=%v", plans, err)
	}
	return plans[0]
}

func verifyReviewMigrationPlan(t *testing.T, store *SQLiteReviewGatewayStore, plan ReviewResourceMigration, chatID string) ReviewResourceMigration {
	t.Helper()
	verified, err := store.VerifyLegacyReviewResourceMigration(
		context.Background(), plan.MigrationID,
		LegacyReviewResourceVerificationInput{
			ExpectedInstallation: plan.InstallationID,
			ExpectedScope:        plan.TargetScope,
			ExpectedChatID:       chatID,
			Method:               "fixture_verified",
			Actor:                "fixture-operator",
			Confirmed:            true,
		},
		fixtureLegacyReviewResourceVerifier{},
		reviewMigrationTestTime.Add(time.Minute),
	)
	if err != nil {
		t.Fatalf("verify migration: %v", err)
	}
	return verified
}

func applyReviewMigrationPlan(t *testing.T, store *SQLiteReviewGatewayStore, plan ReviewResourceMigration) ReviewResourceMigration {
	t.Helper()
	applied, err := store.ApplyLegacyReviewResourceMigration(
		context.Background(), plan.MigrationID, "fixture-operator", reviewMigrationTestTime.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatalf("apply migration: %v", err)
	}
	return applied
}

func enabledMigrationPolicy(installation, resource string, scope ReviewResourceScope) *ReviewResourceScopePolicy {
	return &ReviewResourceScopePolicy{
		InstallationID: installation, ResourceType: resource, TargetScope: scope,
		MigrationEnabled: true, UpdatedBy: "fixture-operator",
	}
}

func TestLegacyMigrationScanCreatesStablePlan(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	if plan.MigrationID == "" || plan.TargetWorkItemKey == "" || plan.Status != ReviewResourceMigrationNeedsReconciliation {
		t.Fatalf("unexpected plan: %#v", plan)
	}
}

func TestLegacyMigrationScanIsIdempotent(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	first := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	if err := store.ScanLegacyReviewResourceMigrations(context.Background(), reviewMigrationTestTime.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	plans, _ := store.ListReviewResourceMigrations(context.Background(), "", "", "")
	if len(plans) != 1 || plans[0].MigrationID != first.MigrationID {
		t.Fatalf("scan duplicated plans: %#v", plans)
	}
}

func TestUnsupportedLegacyResourceNeedsReconciliation(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, "feishu_unknown", nil)
	if plan.Status != ReviewResourceMigrationNeedsReconciliation || plan.ReasonCode != "unsupported_resource_type" {
		t.Fatalf("unsupported plan=%#v", plan)
	}
}

func TestAmbiguousInstallationIsSkipped(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a", "installation-b"}, ReviewResourceCard, nil)
	if plan.Status != ReviewResourceMigrationSkippedAmbiguous || plan.InstallationID != "" {
		t.Fatalf("ambiguous plan=%#v", plan)
	}
}

func TestMissingLegacyChatCannotMigrateCard(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "", []string{"installation-a"}, ReviewResourceCard, nil)
	if plan.Status != ReviewResourceMigrationNeedsReconciliation || plan.ReasonCode != "missing_legacy_chat" {
		t.Fatalf("missing chat plan=%#v", plan)
	}
}

func TestDiscoveryDoesNotCopyRemoteID(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	target, _ := ResolveReviewResourceTarget(ReviewResourceCard, plan.InstallationID, "chat-a", plan.Repository, plan.PRNumber, plan.TargetScope)
	state, err := store.GetReviewResourceState(context.Background(), target, ReviewResourceCard)
	if err != nil || state.RemoteID != "" {
		t.Fatalf("discovery copied remote ID: %#v err=%v", state, err)
	}
}

func TestUnverifiedLegacyCardCannotBeApplied(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	if _, err := store.ApplyLegacyReviewResourceMigration(context.Background(), plan.MigrationID, "operator", reviewMigrationTestTime); !errors.Is(err, ErrReviewResourceMigrationConflict) {
		t.Fatalf("unverified apply err=%v", err)
	}
}

func TestVerifiedLegacyCardAdoptsCanonicalMessageID(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	applyReviewMigrationPlan(t, store, plan)
	card, err := store.GetChatPRPresentation(context.Background(), ReviewGatewayJob{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", PRNumber: 42,
	})
	if err != nil || card.CanonicalMessageID != "remote_legacy_feishu_card" || card.PresentationVersion != 1 || card.CardStatus != "active" {
		t.Fatalf("adopted card=%#v err=%v", card, err)
	}
}

func TestVerifiedLegacyCardDoesNotCreateSecondCard(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	applyReviewMigrationPlan(t, store, plan)
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM chat_pr_presentations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("card count=%d err=%v", count, err)
	}
	if _, err := store.ApplyLegacyReviewResourceMigration(context.Background(), plan.MigrationID, "operator", reviewMigrationTestTime.Add(3*time.Minute)); !errors.Is(err, ErrReviewResourceMigrationConflict) {
		t.Fatalf("second apply err=%v", err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM chat_pr_presentations`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("card count after second apply=%d err=%v", count, err)
	}
}

func TestLegacyCardDoesNotCrossChatBoundary(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	_, err := store.VerifyLegacyReviewResourceMigration(context.Background(), plan.MigrationID,
		LegacyReviewResourceVerificationInput{ExpectedInstallation: "installation-a", ExpectedScope: ReviewResourceScopeChat,
			ExpectedChatID: "chat-b", Method: "fixture_verified", Confirmed: true},
		fixtureLegacyReviewResourceVerifier{}, reviewMigrationTestTime)
	if err == nil {
		t.Fatal("cross-chat verification unexpectedly succeeded")
	}
}

func TestLegacyCardMigrationRetainsSourceMapping(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	applyReviewMigrationPlan(t, store, plan)
	state, err := store.GetReviewResourceState(context.Background(), plan.LegacyWorkItemKey, ReviewResourceCard)
	if err != nil || state.RemoteID == "" {
		t.Fatalf("source mapping lost: %#v err=%v", state, err)
	}
}

func TestLegacyCardDoesNotOverwriteActiveCanonicalCard(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	job := ReviewGatewayJob{InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", PRNumber: 42}
	nowText := reviewMigrationTestTime.Format(time.RFC3339Nano)
	if _, err := store.db.Exec(`INSERT INTO chat_pr_presentations (
		presentation_key, app_scope, installation_id, chat_id, repository, pr_number,
		canonical_message_id, presentation_version, card_status, created_at, updated_at
	) VALUES (?, ?, 'installation-a', 'chat-a', 'owner/repo', 42, 'new-card', 2, 'active', ?, ?)`,
		reviewChatPRPresentationKey(job), reviewGatewayAppScope, nowText, nowText); err != nil {
		t.Fatal(err)
	}
	applied := applyReviewMigrationPlan(t, store, plan)
	if applied.Status != ReviewResourceMigrationSuperseded {
		t.Fatalf("active target outcome=%#v", applied)
	}
	card, _ := store.GetChatPRPresentation(context.Background(), job)
	if card.CanonicalMessageID != "new-card" {
		t.Fatalf("canonical card overwritten: %#v", card)
	}
}

func TestLegacyCardDifferentTargetBecomesSuperseded(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	target, _ := ResolveReviewResourceTarget(ReviewResourceCard, plan.InstallationID, "chat-a", plan.Repository, plan.PRNumber, plan.TargetScope)
	if err := store.SaveReviewResourceState(context.Background(), target, ReviewResourceCard, "different-card", "new", reviewMigrationTestTime); err != nil {
		t.Fatal(err)
	}
	got := applyReviewMigrationPlan(t, store, plan)
	if got.Status != ReviewResourceMigrationSuperseded || got.ReasonCode != "target_resource_already_exists" {
		t.Fatalf("superseded result=%#v", got)
	}
}

func TestBaseMigrationRequiresExplicitPolicy(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceBitable, nil)
	if plan.ReasonCode != "explicit_scope_policy_required" || plan.TargetWorkItemKey != "" {
		t.Fatalf("Base plan=%#v", plan)
	}
}

func TestDocMigrationRequiresExplicitPolicy(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceDoc, nil)
	if plan.ReasonCode != "explicit_scope_policy_required" || plan.TargetWorkItemKey != "" {
		t.Fatalf("Doc plan=%#v", plan)
	}
}

func TestTaskMigrationRequiresVerifiedChat(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceTask, nil)
	if plan.Status != ReviewResourceMigrationNeedsReconciliation {
		t.Fatalf("Task plan=%#v", plan)
	}
	if _, err := store.ApplyLegacyReviewResourceMigration(context.Background(), plan.MigrationID, "operator", reviewMigrationTestTime); err == nil {
		t.Fatal("unverified Task migrated")
	}
}

func TestInstallationScopedBaseCanBeMigrated(t *testing.T) {
	assertProjectionMigration(t, ReviewResourceBitable, ReviewResourceScopeInstallation)
}

func TestInstallationScopedDocCanBeMigrated(t *testing.T) {
	assertProjectionMigration(t, ReviewResourceDoc, ReviewResourceScopeInstallation)
}

func TestChatScopedBaseIsNotShared(t *testing.T) {
	assertChatProjectionKeysDiffer(t, ReviewResourceBitable)
}

func TestChatScopedDocIsNotShared(t *testing.T) {
	assertChatProjectionKeysDiffer(t, ReviewResourceDoc)
}

func TestTaskIsNeverSharedAcrossChats(t *testing.T) {
	assertChatProjectionKeysDiffer(t, ReviewResourceTask)
}

func assertProjectionMigration(t *testing.T, resource string, scope ReviewResourceScope) {
	t.Helper()
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, resource,
		enabledMigrationPolicy("installation-a", resource, scope))
	plan = verifyReviewMigrationPlan(t, store, plan, "chat-a")
	got := applyReviewMigrationPlan(t, store, plan)
	target, _ := ResolveReviewResourceTarget(resource, plan.InstallationID, "chat-a", plan.Repository, plan.PRNumber, plan.TargetScope)
	state, err := store.GetReviewResourceState(context.Background(), target, resource)
	if got.Status != ReviewResourceMigrationMigrated || err != nil || state.RemoteID == "" {
		t.Fatalf("projection migration=%#v state=%#v err=%v", got, state, err)
	}
}

func assertChatProjectionKeysDiffer(t *testing.T, resource string) {
	t.Helper()
	a, errA := ResolveReviewResourceTarget(resource, "installation-a", "chat-a", "owner/repo", 42, ReviewResourceScopeChat)
	b, errB := ResolveReviewResourceTarget(resource, "installation-a", "chat-b", "owner/repo", 42, ReviewResourceScopeChat)
	if errA != nil || errB != nil || a == b {
		t.Fatalf("chat keys a=%q b=%q errA=%v errB=%v", a, b, errA, errB)
	}
}

func TestMigrationCannotApplyBeforeVerification(t *testing.T) {
	TestUnverifiedLegacyCardCannotBeApplied(t)
}

func TestMigrationConditionalTransitionRejectsStaleWorker(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	if _, err := store.transitionReviewResourceMigration(context.Background(), plan.MigrationID,
		ReviewResourceMigrationFailed, ReviewResourceMigrationVerified, "retried", "operator", "", reviewMigrationTestTime); !errors.Is(err, ErrReviewResourceMigrationConflict) {
		t.Fatalf("stale transition err=%v", err)
	}
}

func TestMigrationFailureRollsBackTargetMapping(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceTask, nil), "chat-a")
	if _, err := store.db.Exec(`CREATE TRIGGER fail_review_resource_target BEFORE INSERT ON review_collaboration_resources
		WHEN NEW.work_item_key <> '` + plan.LegacyWorkItemKey + `' BEGIN SELECT RAISE(ABORT, 'fixture failure'); END`); err != nil {
		t.Fatal(err)
	}
	got, err := store.ApplyLegacyReviewResourceMigration(context.Background(), plan.MigrationID, "operator", reviewMigrationTestTime.Add(2*time.Minute))
	if err == nil || got.Status != ReviewResourceMigrationFailed {
		t.Fatalf("failed apply=%#v err=%v", got, err)
	}
	target, _ := ResolveReviewResourceTarget(ReviewResourceTask, plan.InstallationID, "chat-a", plan.Repository, plan.PRNumber, plan.TargetScope)
	state, _ := store.GetReviewResourceState(context.Background(), target, ReviewResourceTask)
	if state.RemoteID != "" {
		t.Fatalf("failed target mapping persisted: %#v", state)
	}
}

func TestMigrationRetryStartsFromFailed(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceTask, nil)
	if _, err := store.db.Exec(`UPDATE review_resource_migrations SET status='failed' WHERE migration_id=?`, plan.MigrationID); err != nil {
		t.Fatal(err)
	}
	got, err := store.RetryLegacyReviewResourceMigration(context.Background(), plan.MigrationID, "operator", reviewMigrationTestTime)
	if err != nil || got.Status != ReviewResourceMigrationVerified {
		t.Fatalf("retry=%#v err=%v", got, err)
	}
}

func TestMigratedPlanCannotBeAppliedTwice(t *testing.T) {
	TestVerifiedLegacyCardDoesNotCreateSecondCard(t)
}

func TestMigrationAuditIsAppendOnly(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil), "chat-a")
	before, _ := store.ListReviewResourceMigrationAudit(context.Background(), plan.MigrationID)
	applyReviewMigrationPlan(t, store, plan)
	after, _ := store.ListReviewResourceMigrationAudit(context.Background(), plan.MigrationID)
	if len(after) <= len(before) {
		t.Fatalf("audit was not appended: before=%d after=%d", len(before), len(after))
	}
	for index := range before {
		if before[index].AuditID != after[index].AuditID {
			t.Fatalf("existing audit changed at %d", index)
		}
	}
	if _, err := store.db.Exec(`UPDATE review_resource_migration_audit SET action='tampered' WHERE migration_id=?`, plan.MigrationID); err == nil {
		t.Fatal("append-only audit accepted UPDATE")
	}
	if _, err := store.db.Exec(`DELETE FROM review_resource_migration_audit WHERE migration_id=?`, plan.MigrationID); err == nil {
		t.Fatal("append-only audit accepted DELETE")
	}
}

func TestMigrationAuditContainsNoRawIdentifiers(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := verifyReviewMigrationPlan(t, store,
		seedReviewMigrationPlan(t, store, "chat-secret", []string{"installation-a"}, ReviewResourceCard, nil), "chat-secret")
	applyReviewMigrationPlan(t, store, plan)
	audits, _ := store.ListReviewResourceMigrationAudit(context.Background(), plan.MigrationID)
	for _, audit := range audits {
		text := string(audit.DetailJSON) + audit.ActorHash
		if strings.Contains(text, "chat-secret") || strings.Contains(text, "remote_legacy") || strings.Contains(text, "fixture-operator") {
			t.Fatalf("raw identifier leaked in audit: %s", text)
		}
	}
}

func TestMigrationPlansSurviveSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	store, _ := OpenSQLiteReviewGatewayStore(path)
	insertLegacyProductInvariantItem(t, store, "chat-a", []string{"installation-a"}, []string{ReviewResourceCard})
	if err := store.ScanLegacyReviewResourceMigrations(context.Background(), reviewMigrationTestTime); err != nil {
		t.Fatal(err)
	}
	plans, _ := store.ListReviewResourceMigrations(context.Background(), "", "", "")
	_ = store.Close()
	store, _ = OpenSQLiteReviewGatewayStore(path)
	defer store.Close()
	reopened, _ := store.ListReviewResourceMigrations(context.Background(), "", "", "")
	if len(plans) != 1 || len(reopened) != 1 || plans[0].MigrationID != reopened[0].MigrationID {
		t.Fatalf("restart plans before=%#v after=%#v", plans, reopened)
	}
}

func TestRestartDoesNotDuplicateMigrationPlans(t *testing.T) {
	TestMigrationPlansSurviveSQLiteRestart(t)
}

func TestTwoChatsKeepIndependentCanonicalCards(t *testing.T) {
	a := reviewChatPRPresentationKey(ReviewGatewayJob{InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", PRNumber: 42})
	b := reviewChatPRPresentationKey(ReviewGatewayJob{InstallationID: "installation-a", ChatID: "chat-b", Repository: "owner/repo", PRNumber: 42})
	if a == b {
		t.Fatal("two chats share canonical card key")
	}
}

func TestTwoChatsKeepIndependentTasks(t *testing.T) {
	assertChatProjectionKeysDiffer(t, ReviewResourceTask)
}

func TestTwoChatsShareInstallationPresentation(t *testing.T) {
	a := reviewPRPresentationKey("installation-a", "owner/repo", 42)
	b := reviewPRPresentationKey("installation-a", "owner/repo", 42)
	if a != b {
		t.Fatalf("installation presentation differs: %q %q", a, b)
	}
}

func TestInstallationScopedProjectionIsShared(t *testing.T) {
	a, errA := ResolveReviewResourceTarget(ReviewResourceDoc, "installation-a", "chat-a", "owner/repo", 42, ReviewResourceScopeInstallation)
	b, errB := ResolveReviewResourceTarget(ReviewResourceDoc, "installation-a", "chat-b", "owner/repo", 42, ReviewResourceScopeInstallation)
	if errA != nil || errB != nil || a != b {
		t.Fatalf("installation projections a=%q b=%q errA=%v errB=%v", a, b, errA, errB)
	}
}

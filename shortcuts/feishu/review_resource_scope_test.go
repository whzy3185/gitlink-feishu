package feishu

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestCardScopeIsAlwaysChat(t *testing.T) {
	want := stableKey("review-work-item", reviewCollaborationScopeKey("installation-a", "chat-a", "owner/repo", 42))
	got, err := ResolveReviewResourceTarget(ReviewResourceCard, "installation-a", "chat-a", "owner/repo", 42, ReviewResourceScopeInstallation)
	if err != nil || got != want {
		t.Fatalf("card target=%q err=%v want=%q", got, err, want)
	}
}

func TestTaskScopeIsAlwaysChat(t *testing.T) {
	want := stableKey("review-work-item", reviewCollaborationScopeKey("installation-a", "chat-a", "owner/repo", 42))
	got, err := ResolveReviewResourceTarget(ReviewResourceTask, "installation-a", "chat-a", "owner/repo", 42, ReviewResourceScopeInstallation)
	if err != nil || got != want {
		t.Fatalf("task target=%q err=%v want=%q", got, err, want)
	}
}

func TestBaseChatScopeUsesCollaborationKey(t *testing.T) {
	assertReviewResourceTarget(t, ReviewResourceBitable, ReviewResourceScopeChat, "chat-a", reviewCollaborationScopeKey("installation-a", "chat-a", "owner/repo", 42))
}

func TestBaseInstallationScopeUsesSnapshotKey(t *testing.T) {
	assertReviewResourceTarget(t, ReviewResourceBitable, ReviewResourceScopeInstallation, "chat-a", reviewSnapshotScopeKey("installation-a", "owner/repo", 42))
}

func TestDocChatScopeUsesCollaborationKey(t *testing.T) {
	assertReviewResourceTarget(t, ReviewResourceDoc, ReviewResourceScopeChat, "chat-a", reviewCollaborationScopeKey("installation-a", "chat-a", "owner/repo", 42))
}

func TestDocInstallationScopeUsesSnapshotKey(t *testing.T) {
	assertReviewResourceTarget(t, ReviewResourceDoc, ReviewResourceScopeInstallation, "chat-a", reviewSnapshotScopeKey("installation-a", "owner/repo", 42))
}

func TestDisabledResourceHasNoMigrationTarget(t *testing.T) {
	if target, err := ResolveReviewResourceTarget(ReviewResourceDoc, "installation-a", "chat-a", "owner/repo", 42, ReviewResourceScopeDisabled); target != "" || !errors.Is(err, ErrReviewResourceProjectionDisabled) {
		t.Fatalf("disabled target=%q err=%v", target, err)
	}
}

func TestMissingLegacyPolicyRequiresReconciliation(t *testing.T) {
	if target, err := ResolveReviewResourceTarget(ReviewResourceBitable, "installation-a", "chat-a", "owner/repo", 42, ""); target != "" || !errors.Is(err, ErrReviewResourcePolicyRequired) {
		t.Fatalf("missing policy target=%q err=%v", target, err)
	}
}

func TestMigrationPoliciesSurviveSQLiteRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "scope-policy.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := store.SetReviewResourceScopePolicy(context.Background(), ReviewResourceScopePolicy{
		InstallationID: "installation-a", ResourceType: ReviewResourceDoc,
		TargetScope: ReviewResourceScopeInstallation, MigrationEnabled: true, UpdatedBy: "operator-a",
	}, time.Date(2026, 8, 4, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.GetReviewResourceScopePolicy(context.Background(), "installation-a", ReviewResourceDoc)
	if err != nil || got.TargetScope != want.TargetScope || !got.MigrationEnabled || got.Revision != 1 {
		t.Fatalf("reopened policy=%#v err=%v", got, err)
	}
}

func assertReviewResourceTarget(t *testing.T, resource string, scope ReviewResourceScope, chatID, innerKey string) {
	t.Helper()
	want := stableKey("review-work-item", innerKey)
	got, err := ResolveReviewResourceTarget(resource, "installation-a", chatID, "owner/repo", 42, scope)
	if err != nil || got != want {
		t.Fatalf("%s/%s target=%q err=%v want=%q", resource, scope, got, err, want)
	}
}

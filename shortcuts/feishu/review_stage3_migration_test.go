package feishu

import (
	"context"
	"path/filepath"
	"testing"
)

func requireReviewStageThreeMigration(t *testing.T, version int, name string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "stage-three-migration.db")
	for pass := 0; pass < 2; pass++ {
		store, err := OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("open pass %d: %v", pass+1, err)
		}
		var count int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=? AND name=?`, version, name).Scan(&count); err != nil {
			_ = store.Close()
			t.Fatal(err)
		}
		_ = store.Close()
		if count != 1 {
			t.Fatalf("migration %d/%s count=%d on pass %d", version, name, count, pass+1)
		}
	}
}

func TestSubscriptionMigrationIsIdempotent(t *testing.T) {
	requireReviewStageThreeMigration(t, 7, "review_chat_subscriptions_v1")
}

func TestEventInboxMigrationIsIdempotent(t *testing.T) {
	requireReviewStageThreeMigration(t, 8, "review_event_inbox_routes_v1")
}

func TestWebhookPolicyMigrationIsIdempotent(t *testing.T) {
	requireReviewStageThreeMigration(t, 9, "gitlink_webhook_security_policy_v1")
}

func TestReconciliationMigrationIsIdempotent(t *testing.T) {
	requireReviewStageThreeMigration(t, 10, "review_reconciliation_cursors_v1")
}

func TestStageThreeMigrationRollbackLeavesDatabaseUsable(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "rollback.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	err = applyReviewGatewaySchemaMigrations(store.db, []reviewGatewaySchemaMigration{{
		Version: 99, Name: "intentional_failure",
		Statements: []string{`INSERT INTO table_that_does_not_exist(value) VALUES(1)`},
	}})
	if err == nil {
		t.Fatal("invalid migration unexpectedly succeeded")
	}
	var versionCount, tableCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=99`).Scan(&versionCount); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_event_inbox`).Scan(&tableCount); err != nil {
		t.Fatalf("database unusable after migration rollback: %v", err)
	}
	if versionCount != 0 || tableCount != 0 {
		t.Fatalf("failed migration leaked state: version=%d rows=%d", versionCount, tableCount)
	}
}

func TestExistingSubscriptionsAreNotAutomaticallyEnabled(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_chat_subscriptions WHERE enabled=1`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("configuration sync implicitly enabled %d subscriptions", count)
	}
	if subscriptions, err := store.ListReviewChatSubscriptions(context.Background(), "", "", "", false); err != nil || len(subscriptions) != 0 {
		t.Fatalf("implicit subscriptions=%+v err=%v", subscriptions, err)
	}
}

package feishu

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

var reviewSubscriptionTestTime = time.Date(2026, 8, 4, 9, 0, 0, 0, time.UTC)

func newReviewSubscriptionTestStore(t *testing.T, bindings []ReviewChatBinding) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "subscriptions.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	configuration := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "installation-a",
			GitLinkHost:         "https://www.gitlink.org.cn/",
			Owner:               "owner",
			OperationMode:       "collaborate",
			AllowedRepositories: []string{"owner/repo", "owner/second"},
			Enabled:             true,
		}},
		Bindings: bindings,
	}
	if err := store.SyncReviewGatewayConfiguration(context.Background(), configuration, "subscription-test", reviewSubscriptionTestTime); err != nil {
		t.Fatalf("SyncReviewGatewayConfiguration: %v", err)
	}
	return store
}

func reviewSubscriptionBinding(chatID string, enabled bool, administrators ...string) ReviewChatBinding {
	return ReviewChatBinding{
		ChatID: chatID, InstallationID: "installation-a",
		Repositories:      []string{"owner/repo", "owner/second"},
		DefaultRepository: "owner/repo", Enabled: enabled,
		AdminUserIDs: administrators,
	}
}

func addReviewTestSubscription(t *testing.T, store *SQLiteReviewGatewayStore, chatID, actor string) ReviewChatSubscription {
	t.Helper()
	subscription, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: chatID, Repository: "owner/repo", ActorID: actor,
		AddGroups: []string{ReviewSubscriptionGroupPulls, ReviewSubscriptionGroupReviews},
	}, reviewSubscriptionTestTime)
	if err != nil {
		t.Fatalf("ChangeReviewChatSubscription: %v", err)
	}
	return subscription
}

func TestBindingDoesNotImplicitlyCreateSubscription(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	subscriptions, err := store.ListReviewChatSubscriptions(context.Background(), "", "", "", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 0 {
		t.Fatalf("binding implicitly created %d subscriptions", len(subscriptions))
	}
}

func TestChatAdminCanCreateSubscription(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	subscription := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	if !subscription.Enabled || subscription.Revision != 1 || len(subscription.EventGroups) != 2 {
		t.Fatalf("unexpected subscription: %+v", subscription)
	}
}

func TestNonAdminCannotCreateSubscription(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "member-a",
		AddGroups: []string{ReviewSubscriptionGroupPulls},
	}, reviewSubscriptionTestTime)
	if err == nil {
		t.Fatal("non-administrator created a subscription")
	}
}

func TestNonAdminCannotModifySubscription(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	current := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "member-a",
		RemoveGroups: []string{ReviewSubscriptionGroupReviews}, ExpectedRevision: current.Revision,
	}, reviewSubscriptionTestTime.Add(time.Minute))
	if err == nil {
		t.Fatal("non-administrator modified a subscription")
	}
}

func TestSubscriptionRequiresEnabledBinding(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", false, "admin-a")})
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupPulls},
	}, reviewSubscriptionTestTime)
	if err == nil {
		t.Fatal("disabled binding allowed a subscription")
	}
}

func TestSubscriptionRequiresAllowedRepository(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	if _, err := store.db.Exec(`DELETE FROM installation_repositories WHERE installation_id=? AND repository=?`, "installation-a", "owner/repo"); err != nil {
		t.Fatal(err)
	}
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupPulls},
	}, reviewSubscriptionTestTime)
	if err == nil {
		t.Fatal("installation-unauthorized repository allowed a subscription")
	}
}

func TestRepeatedSubscriptionIsIdempotent(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	first := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	second := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	if first.SubscriptionID != second.SubscriptionID || second.Revision != first.Revision {
		t.Fatalf("repeated subscription changed state: first=%+v second=%+v", first, second)
	}
}

func TestSubscriptionRevisionPreventsLostUpdate(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	current := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	updated, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupThreads}, ExpectedRevision: current.Revision,
	}, reviewSubscriptionTestTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		AddGroups: []string{ReviewSubscriptionGroupMerge}, ExpectedRevision: current.Revision,
	}, reviewSubscriptionTestTime.Add(2*time.Minute))
	if !errors.Is(err, ErrReviewSubscriptionConflict) || updated.Revision != current.Revision+1 {
		t.Fatalf("stale revision result: updated=%+v err=%v", updated, err)
	}
}

func TestTwoChatsHaveIndependentSubscriptions(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{
		reviewSubscriptionBinding("chat-a", true, "admin-a"),
		reviewSubscriptionBinding("chat-b", true, "admin-b"),
	})
	first := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	second, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-b", Repository: "owner/repo", ActorID: "admin-b",
		AddGroups: []string{ReviewSubscriptionGroupMerge},
	}, reviewSubscriptionTestTime)
	if err != nil {
		t.Fatal(err)
	}
	if first.SubscriptionID == second.SubscriptionID || first.ChatIDHash == second.ChatIDHash || len(second.EventGroups) != 1 {
		t.Fatalf("chat subscriptions are not isolated: first=%+v second=%+v", first, second)
	}
}

func TestDisabledSubscriptionReceivesNoRoutes(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	current := addReviewTestSubscription(t, store, "chat-a", "admin-a")
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		RemoveGroups: current.EventGroups, ExpectedRevision: current.Revision,
	}, reviewSubscriptionTestTime.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	routes, err := store.ListReviewChatSubscriptions(context.Background(), "installation-a", "", "owner/repo", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 0 {
		t.Fatalf("disabled subscription remained routable: %+v", routes)
	}
}

func TestSetDefaultRepositoryIsTransactional(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	if err := store.SetDefaultReviewRepository(context.Background(), "installation-a", "chat-a", "owner/second", "admin-a"); err != nil {
		t.Fatal(err)
	}
	assertDefault := func(want string) {
		t.Helper()
		var got string
		if err := store.db.QueryRow(`SELECT repository FROM chat_repository_bindings
			WHERE installation_id=? AND chat_id=? AND is_default=1`, "installation-a", "chat-a").Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("default repository = %q, want %q", got, want)
		}
	}
	assertDefault("owner/second")
	if err := store.SetDefaultReviewRepository(context.Background(), "installation-a", "chat-a", "owner/missing", "admin-a"); err == nil {
		t.Fatal("missing repository was accepted as default")
	}
	assertDefault("owner/second")
}

func TestNotificationModeValidation(t *testing.T) {
	store := newReviewSubscriptionTestStore(t, []ReviewChatBinding{reviewSubscriptionBinding("chat-a", true, "admin-a")})
	_, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{
		InstallationID: "installation-a", ChatID: "chat-a", Repository: "owner/repo", ActorID: "admin-a",
		NotificationMode: "broadcast_everything",
	}, reviewSubscriptionTestTime)
	if err == nil {
		t.Fatal("invalid notification mode was accepted")
	}
}

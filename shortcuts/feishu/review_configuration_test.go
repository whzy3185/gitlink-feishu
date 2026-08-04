package feishu

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var reviewConfigurationTestTime = time.Date(2026, 8, 5, 10, 0, 0, 0, time.UTC)

func openReviewConfigurationTestStore(t *testing.T) *SQLiteReviewGatewayStore {
	t.Helper()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "configuration.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func reviewConfigurationFixture() ReviewGatewayBindings {
	return ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID: "installation-a", GitLinkHost: "https://www.gitlink.org.cn/", Owner: "owner",
			CredentialRef: "env:GITLINK_SECRET_REFERENCE", OperationMode: "write",
			AllowedRepositories: []string{"owner/two", "owner/one"}, AllowPublicRead: true,
			WebhookSecretRef: "env:GITLINK_WEBHOOK_REFERENCE", Enabled: true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID: "oc_configuration", InstallationID: "installation-a", Repositories: []string{"owner/two", "owner/one"},
			DefaultRepository: "owner/one", AllowPublicRead: true, Enabled: true,
			AdminUserIDs: []string{"ou_admin"}, AllowedUserIDs: []string{"ou_user"},
		}},
		IdentityBindings: []ReviewIdentityBinding{{InstallationID: "installation-a", FeishuUserID: "ou_user", GitLinkLogin: "gitlink-user", VerificationMethod: "admin_config", Enabled: true}},
	}
}

func reviewConfigurationOptions() ReviewConfigurationApplyOptions {
	config := DefaultReviewServiceConfig()
	config.StateDB = `E:\private\configuration.db`
	config.BackupDirectory = `E:\private\backups`
	config.AdminTokenRef = "env:ADMIN_TOKEN_REFERENCE"
	return ReviewConfigurationApplyOptions{Source: `E:\private\bindings.json`, ActorID: "ou_admin", WorkerConfig: map[string]int{"gitlink_read": 3}, LimitConfig: map[string]int{"queue_size": 128}, ServiceConfig: config}
}

func applyReviewConfigurationFixture(t *testing.T, store *SQLiteReviewGatewayStore, bindings ReviewGatewayBindings, options ReviewConfigurationApplyOptions, now time.Time) ReviewConfigurationState {
	t.Helper()
	state, _, err := store.ApplyReviewGatewayConfiguration(context.Background(), bindings, options, now)
	if err != nil {
		t.Fatal(err)
	}
	return state
}

func TestConfigurationFingerprintIsDeterministic(t *testing.T) {
	firstStore := openReviewConfigurationTestStore(t)
	first := applyReviewConfigurationFixture(t, firstStore, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	secondStore := openReviewConfigurationTestStore(t)
	reordered := reviewConfigurationFixture()
	reordered.Installations[0].AllowedRepositories = []string{"owner/one", "owner/two", "owner/one"}
	reordered.Bindings[0].Repositories = []string{"owner/one", "owner/two"}
	second := applyReviewConfigurationFixture(t, secondStore, reordered, reviewConfigurationOptions(), reviewConfigurationTestTime)
	if first.ConfigFingerprint != second.ConfigFingerprint {
		t.Fatalf("fingerprints differ: %s %s", first.ConfigFingerprint, second.ConfigFingerprint)
	}
}

func TestConfigurationFingerprintDoesNotContainSecrets(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	state := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	var workerJSON, limitJSON, serviceJSON, source string
	if err := store.db.QueryRow(`SELECT worker_config_json,limit_config_json,service_config_json,source FROM review_configuration_revisions WHERE config_revision=1`).Scan(&workerJSON, &limitJSON, &serviceJSON, &source); err != nil {
		t.Fatal(err)
	}
	all := strings.Join([]string{state.ConfigFingerprint, workerJSON, limitJSON, serviceJSON, source}, "\n")
	for _, forbidden := range []string{"GITLINK_SECRET", "GITLINK_WEBHOOK", "ADMIN_TOKEN", `E:\private`, "ou_admin", "ou_user"} {
		if strings.Contains(all, forbidden) {
			t.Fatalf("configuration revision exposed %q: %s", forbidden, all)
		}
	}
}

func TestConfigurationSourceUsesCrossPlatformBaseName(t *testing.T) {
	for _, test := range []struct {
		name   string
		source string
		want   string
	}{
		{name: "windows", source: `E:\private\bindings.json`, want: "bindings.json"},
		{name: "unix", source: `/private/bindings.json`, want: "bindings.json"},
		{name: "windows root", source: `E:\\`, want: "file"},
		{name: "unix root", source: `/`, want: "file"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, hash := reviewConfigurationSource(test.source)
			if got != test.want || hash == "" {
				t.Fatalf("source=%q hash=%q want=%q", got, hash, test.want)
			}
			if strings.Contains(got, "private") {
				t.Fatalf("source leaked parent directory: %q", got)
			}
		})
	}
}

func TestSameConfigurationDoesNotIncrementRevision(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	first := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	second := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime.Add(time.Minute))
	if first.ConfigRevision != 1 || second.ConfigRevision != 1 {
		t.Fatalf("revisions=%d,%d", first.ConfigRevision, second.ConfigRevision)
	}
}

func TestChangedConfigurationIncrementsRevision(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	first := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	changed := reviewConfigurationFixture()
	changed.Bindings[0].AllowedUserIDs = append(changed.Bindings[0].AllowedUserIDs, "ou_second")
	second := applyReviewConfigurationFixture(t, store, changed, reviewConfigurationOptions(), reviewConfigurationTestTime.Add(time.Minute))
	if second.ConfigRevision != first.ConfigRevision+1 || second.ConfigFingerprint == first.ConfigFingerprint {
		t.Fatalf("states=%#v %#v", first, second)
	}
}

func TestConfigurationRevisionIsTransactional(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	first := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	if _, err := store.db.Exec(`CREATE TRIGGER reject_configuration_revision BEFORE INSERT ON review_configuration_revisions BEGIN SELECT RAISE(ABORT, 'test rollback'); END`); err != nil {
		t.Fatal(err)
	}
	changed := reviewConfigurationFixture()
	changed.Bindings[0].AllowedUserIDs = append(changed.Bindings[0].AllowedUserIDs, "ou_second")
	if _, _, err := store.ApplyReviewGatewayConfiguration(context.Background(), changed, reviewConfigurationOptions(), reviewConfigurationTestTime.Add(time.Minute)); err == nil {
		t.Fatal("failed transaction was accepted")
	}
	current, err := store.CurrentReviewConfigurationState(context.Background())
	if err != nil || current.ConfigRevision != first.ConfigRevision || current.ConfigFingerprint != first.ConfigFingerprint {
		t.Fatalf("transaction leaked state=%#v err=%v", current, err)
	}
}

func TestExpectedRevisionPreventsLostUpdate(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	first := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	expected := first.ConfigRevision
	changed := reviewConfigurationFixture()
	changed.Bindings[0].AllowedUserIDs = append(changed.Bindings[0].AllowedUserIDs, "ou_second")
	options := reviewConfigurationOptions()
	options.ExpectedRevision = &expected
	_ = applyReviewConfigurationFixture(t, store, changed, options, reviewConfigurationTestTime.Add(time.Minute))
	stale := changed
	stale.Bindings[0].AllowedUserIDs = append(stale.Bindings[0].AllowedUserIDs, "ou_third")
	if _, _, err := store.ApplyReviewGatewayConfiguration(context.Background(), stale, options, reviewConfigurationTestTime.Add(2*time.Minute)); !errors.Is(err, ErrReviewConfigurationConflict) {
		t.Fatalf("stale update err=%v", err)
	}
}

func TestInstallationRevisionPreventsLostUpdate(t *testing.T) {
	testReviewEntityRevisionConflict(t, "gitlink_installations", "installation_id=?", []interface{}{"installation-a"})
}

func TestBindingRevisionPreventsLostUpdate(t *testing.T) {
	testReviewEntityRevisionConflict(t, "chat_repository_bindings", "chat_id=? AND installation_id=? AND repository=?", []interface{}{"oc_configuration", "installation-a", "owner/one"})
}

func TestIdentityRevisionPreventsLostUpdate(t *testing.T) {
	testReviewEntityRevisionConflict(t, "review_identity_bindings", "installation_id=? AND feishu_user_id=?", []interface{}{"installation-a", "ou_user"})
}

func testReviewEntityRevisionConflict(t *testing.T, table, where string, args []interface{}) {
	t.Helper()
	store := openReviewConfigurationTestStore(t)
	applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	if revision, err := store.CompareAndSwapReviewEntityRevision(context.Background(), table, where, args, 1, "first", reviewConfigurationTestTime.Add(time.Minute)); err != nil || revision != 2 {
		t.Fatalf("first revision=%d err=%v", revision, err)
	}
	if _, err := store.CompareAndSwapReviewEntityRevision(context.Background(), table, where, args, 1, "stale", reviewConfigurationTestTime.Add(2*time.Minute)); !errors.Is(err, ErrReviewConfigurationConflict) {
		t.Fatalf("stale entity update err=%v", err)
	}
}

func TestSubscriptionRevisionContinuesToWork(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	created, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{InstallationID: "installation-a", ChatID: "oc_configuration", Repository: "owner/one", ActorID: "ou_admin", AddGroups: []string{ReviewSubscriptionGroupReviews}}, reviewConfigurationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := store.ChangeReviewChatSubscription(context.Background(), ReviewSubscriptionChange{InstallationID: "installation-a", ChatID: "oc_configuration", Repository: "owner/one", ActorID: "ou_admin", AddGroups: []string{ReviewSubscriptionGroupThreads}, ExpectedRevision: created.Revision}, reviewConfigurationTestTime.Add(time.Minute))
	if err != nil || updated.Revision != created.Revision+1 {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
}

func TestResourcePolicyRevisionContinuesToWork(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	created, err := store.SetReviewResourceScopePolicy(context.Background(), ReviewResourceScopePolicy{InstallationID: "installation-a", ResourceType: ReviewResourceDoc, TargetScope: ReviewResourceScopeChat, UpdatedBy: "first"}, reviewConfigurationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	updated := created
	updated.TargetScope = ReviewResourceScopeInstallation
	updated.UpdatedBy = "second"
	updated, err = store.SetReviewResourceScopePolicy(context.Background(), updated, reviewConfigurationTestTime.Add(time.Minute))
	if err != nil || updated.Revision != 2 {
		t.Fatalf("updated=%#v err=%v", updated, err)
	}
	created.TargetScope = ReviewResourceScopeDisabled
	if _, err := store.SetReviewResourceScopePolicy(context.Background(), created, reviewConfigurationTestTime.Add(2*time.Minute)); !errors.Is(err, ErrReviewResourcePolicyConflict) {
		t.Fatalf("stale resource policy err=%v", err)
	}
}

func TestConfigurationRevisionAuditIsAppendOnly(t *testing.T) {
	store := openReviewConfigurationTestStore(t)
	applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	if _, err := store.db.Exec(`UPDATE review_configuration_revisions SET source='changed' WHERE config_revision=1`); err == nil {
		t.Fatal("configuration revision update succeeded")
	}
	if _, err := store.db.Exec(`DELETE FROM review_configuration_revisions WHERE config_revision=1`); err == nil {
		t.Fatal("configuration revision delete succeeded")
	}
}

func TestConfigurationRevisionSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "restart.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	want := applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	got, err := store.CurrentReviewConfigurationState(context.Background())
	if err != nil || got.ConfigRevision != want.ConfigRevision || got.ConfigFingerprint != want.ConfigFingerprint {
		t.Fatalf("got=%#v want=%#v err=%v", got, want, err)
	}
}

func TestConfigurationRevisionMigrationIsIdempotent(t *testing.T) {
	testReviewConfigurationMigrationIdempotent(t, 17)
}

func TestEntityRevisionMigrationIsIdempotent(t *testing.T) {
	testReviewConfigurationMigrationIdempotent(t, 19)
}

func testReviewConfigurationMigrationIdempotent(t *testing.T, version int) {
	t.Helper()
	store := openReviewConfigurationTestStore(t)
	if err := applyReviewGatewaySchemaMigrations(store.db, reviewGatewaySchemaMigrations); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version=?`, version).Scan(&count); err != nil || count != 1 {
		t.Fatalf("version=%d count=%d err=%v", version, count, err)
	}
}

func TestExistingEntityRevisionIsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "preserved.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	applyReviewConfigurationFixture(t, store, reviewConfigurationFixture(), reviewConfigurationOptions(), reviewConfigurationTestTime)
	if _, err := store.db.Exec(`UPDATE gitlink_installations SET revision=7 WHERE installation_id='installation-a'`); err != nil {
		t.Fatal(err)
	}
	_ = store.Close()
	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var revision int
	if err := store.db.QueryRow(`SELECT revision FROM gitlink_installations WHERE installation_id='installation-a'`).Scan(&revision); err != nil || revision != 7 {
		t.Fatalf("revision=%d err=%v", revision, err)
	}
}

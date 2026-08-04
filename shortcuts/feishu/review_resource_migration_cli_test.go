package feishu

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestReviewMigrationCLIRequiresExplicitOperatorVerification(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	plan := seedReviewMigrationPlan(t, store, "chat-a", []string{"installation-a"}, ReviewResourceCard, nil)
	_, err := executeReviewMigrationCommand(context.Background(), store, ReviewMigrationCommandOptions{
		Action: "verify", MigrationID: plan.MigrationID, InstallationID: plan.InstallationID,
		Scope: plan.TargetScope, ChatID: "chat-a", Method: "fixture-verified",
		Actor: "operator", Confirmed: true,
	}, reviewMigrationTestTime)
	if err == nil || !strings.Contains(err.Error(), "operator-confirmed") {
		t.Fatalf("CLI accepted fixture verification: %v", err)
	}
}

func TestReviewMigrationCLIListIsRedacted(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	seedReviewMigrationPlan(t, store, "chat-secret", []string{"installation-a"}, ReviewResourceCard, nil)
	result, err := executeReviewMigrationCommand(context.Background(), store,
		ReviewMigrationCommandOptions{Action: "list"}, reviewMigrationTestTime)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := renderReviewMigrationCommandResult(&output, result, "json"); err != nil {
		t.Fatal(err)
	}
	text := output.String()
	if strings.Contains(text, "chat-secret") || strings.Contains(text, "remote_legacy") {
		t.Fatalf("CLI output leaked raw identifier: %s", text)
	}
}

func TestReviewMigrationCLIPolicySetAndList(t *testing.T) {
	store := newReviewMigrationTestStore(t)
	set, err := executeReviewMigrationCommand(context.Background(), store, ReviewMigrationCommandOptions{
		Action: "policy-set", InstallationID: "installation-a", ResourceType: ReviewResourceDoc,
		Scope: ReviewResourceScopeInstallation, EnableMigration: true, Actor: "operator-secret",
	}, reviewMigrationTestTime)
	if err != nil || len(set.Policies) != 1 || !set.Policies[0].MigrationEnabled {
		t.Fatalf("policy set=%#v err=%v", set, err)
	}
	listed, err := executeReviewMigrationCommand(context.Background(), store,
		ReviewMigrationCommandOptions{Action: "policy-list"}, reviewMigrationTestTime)
	if err != nil || len(listed.Policies) != 1 || listed.Policies[0].UpdatedByHash == "operator-secret" {
		t.Fatalf("policy list=%#v err=%v", listed, err)
	}
}

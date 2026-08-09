package feishu

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type staticReviewDataProvider struct{ data ReviewData }

func (p staticReviewDataProvider) FetchReviewData(context.Context, *common.RuntimeContext, ReviewDataRequest) (ReviewData, error) {
	return p.data, nil
}

type staticReviewCollaboratorReader struct {
	collaborators []ReviewRepositoryCollaborator
}

func (r staticReviewCollaboratorReader) ListRepositoryCollaborators(context.Context, *common.RuntimeContext, string, string) ([]ReviewRepositoryCollaborator, error) {
	return append([]ReviewRepositoryCollaborator(nil), r.collaborators...), nil
}

type staticFeishuDisplayNameResolver string

func (r staticFeishuDisplayNameResolver) ResolveFeishuDisplayName(context.Context, string) (string, error) {
	return string(r), nil
}

func TestParseReviewCollaborationCommands(t *testing.T) {
	tests := []struct {
		input, action, argument string
	}{
		{"领取 owner/repo PR #42", "claim_review", ""},
		{"取消领取 owner/repo PR #42", "release_review", ""},
		{"释放 owner/repo PR #42", "release_review", ""},
		{"设置 owner/repo PR #42 审查截止 2026-08-10", "set_review_deadline", "2026-08-10"},
		{"设置 owner/repo PR #42 截止 2026-08-10", "set_review_deadline", "2026-08-10"},
		{"清除 owner/repo PR #42 审查截止", "clear_review_deadline", ""},
	}
	for _, test := range tests {
		intent := parseReviewGatewayIntent(test.input)
		if intent.Name != test.action || intent.Repository != "owner/repo" || intent.PRNumber != 42 || intent.Argument != test.argument {
			t.Fatalf("parse %q = %#v", test.input, intent)
		}
	}
}

func TestReviewGatewayClaimRequiresIdentityButNotAllowedUserList(t *testing.T) {
	now := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)
	bindings := ReviewGatewayBindings{
		Bindings:         []ReviewChatBinding{{ChatID: "oc_fixture", Repository: "owner/repo", Enabled: true}},
		IdentityBindings: []ReviewIdentityBinding{{FeishuUserID: "ou_alice", GitLinkLogin: "alice", Enabled: true}},
	}
	gateway, err := NewReviewGateway(bindings, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	receipt, err := gateway.Plan(ReviewGatewayEvent{
		MessageID: "om_claim", EventType: "message", ChatID: "oc_fixture", UserID: "ou_alice",
		Content: "领取 owner/repo PR #42", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !receipt.Accepted || receipt.Job == nil || !receipt.Job.CollaborationAuthorized || receipt.Job.GitLinkLogin != "alice" {
		t.Fatalf("claim receipt = %#v, err=%v", receipt, err)
	}
	denied, err := gateway.Plan(ReviewGatewayEvent{
		MessageID: "om_denied", EventType: "message", ChatID: "oc_fixture", UserID: "ou_unbound",
		Content: "领取 owner/repo PR #42", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || denied.Accepted || denied.Reason != "collaboration_identity_required" {
		t.Fatalf("denied receipt = %#v, err=%v", denied, err)
	}
}

func TestReviewCollaborationClaimConflictDeadlineAndRelease(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC)
	base := ReviewGatewayJob{
		JobID: "job-claim", Action: "claim_review", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", RequestedDisplayName: "Alice", GitLinkLogin: "alice",
	}
	claimed, err := store.ApplyCollaborationAction(context.Background(), base, now)
	if err != nil || claimed.AssignedDisplayName != "Alice" || claimed.ActionOutcome != "claimed" {
		t.Fatalf("claimed = %#v, err=%v", claimed, err)
	}
	repeated := base
	repeated.JobID = "job-repeat"
	if item, err := store.ApplyCollaborationAction(context.Background(), repeated, now.Add(time.Minute)); err != nil || item.ActionOutcome != "already_claimed_by_self" {
		t.Fatalf("repeated = %#v, err=%v", item, err)
	}
	other := base
	other.JobID, other.RequestedBy, other.RequestedDisplayName = "job-other", "ou_bob", "Bob"
	if _, err := store.ApplyCollaborationAction(context.Background(), other, now.Add(2*time.Minute)); err == nil || !strings.Contains(err.Error(), "Alice") {
		t.Fatalf("conflict error = %v", err)
	}
	deadline := base
	deadline.JobID, deadline.Action, deadline.Argument = "job-deadline", "set_review_deadline", "2026-08-10"
	item, err := store.ApplyCollaborationAction(context.Background(), deadline, now.Add(3*time.Minute))
	if err != nil || item.DueAt != "2026-08-10" || item.ActionOutcome != "deadline_set" {
		t.Fatalf("deadline = %#v, err=%v", item, err)
	}
	release := base
	release.JobID, release.Action = "job-release", "release_review"
	item, err = store.ApplyCollaborationAction(context.Background(), release, now.Add(4*time.Minute))
	if err != nil || item.AssignedTo != "" || item.DueAt != "2026-08-10" {
		t.Fatalf("release = %#v, err=%v", item, err)
	}
}

func TestReviewGatewayExecutorClaimsForRepositoryCollaborator(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "gateway.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	public := true
	executor := ReviewGatewayExecutor{
		Runtime:       &common.RuntimeContext{},
		DataProvider:  staticReviewDataProvider{data: ReviewData{Repository: "owner/repo", PullRequest: 42, State: "open", RepositoryPublic: &public}},
		Collaboration: store,
		Collaborators: staticReviewCollaboratorReader{collaborators: []ReviewRepositoryCollaborator{{Login: "alice"}}},
		DisplayNames:  staticFeishuDisplayNameResolver("Alice Example"),
		Now:           func() time.Time { return time.Date(2026, 8, 9, 8, 0, 0, 0, time.UTC) },
	}
	result, err := executor.Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-claim", Action: "claim_review", Repository: "owner/repo", PRNumber: 42,
		ChatID: "oc_fixture", RequestedBy: "ou_alice", GitLinkLogin: "alice", CollaborationAuthorized: true,
	})
	if err != nil || result.Collaboration == nil || result.Collaboration.AssignedDisplayName != "Alice Example" || result.MutatesGitLink {
		t.Fatalf("result = %#v, err=%v", result, err)
	}
}

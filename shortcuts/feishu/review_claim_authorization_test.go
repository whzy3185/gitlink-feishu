package feishu

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestReviewGatewayClaimRequiresCollaborationScopeAndIdentity(t *testing.T) {
	now := time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		allowed    []string
		admins     []string
		identities []ReviewIdentityBinding
		user       string
		wantReason string
		wantAdmin  bool
	}{
		{
			name: "allowed reviewer with identity", allowed: []string{"ou_alice"}, user: "ou_alice",
			identities: []ReviewIdentityBinding{{InstallationID: "test", FeishuUserID: "ou_alice", GitLinkLogin: "alice", Enabled: true}},
		},
		{
			name: "allowed reviewer without identity", allowed: []string{"ou_alice"}, user: "ou_alice",
			wantReason: "collaboration_identity_required",
		},
		{
			name: "identity is admitted without AllowedUserIDs", user: "ou_reader",
			identities: []ReviewIdentityBinding{{InstallationID: "test", FeishuUserID: "ou_reader", GitLinkLogin: "reader", Enabled: true}},
		},
		{
			name: "binding admin with identity", admins: []string{"ou_admin"}, user: "ou_admin", wantAdmin: true,
			identities: []ReviewIdentityBinding{{InstallationID: "test", FeishuUserID: "ou_admin", GitLinkLogin: "admin", Enabled: true}},
		},
	}
	for index, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gateway, err := NewReviewGateway(ReviewGatewayBindings{
				SchemaVersion: reviewGatewayBindingSchema,
				Installations: []GitLinkInstallation{{
					InstallationID: "test", GitLinkHost: "https://www.gitlink.org.cn",
					OperationMode: "collaborate", AllowedRepositories: []string{"owner/repo"}, Enabled: true,
				}},
				Bindings: []ReviewChatBinding{{
					ChatID: "oc_review", InstallationID: "test", Repositories: []string{"owner/repo"},
					DefaultRepository: "owner/repo", Enabled: true,
					AllowedUserIDs: test.allowed, AdminUserIDs: test.admins,
				}},
				IdentityBindings: test.identities,
			}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
			if err != nil {
				t.Fatalf("NewReviewGateway: %v", err)
			}
			receipt, err := gateway.Plan(ReviewGatewayEvent{
				MessageID: "om_claim_" + string(rune('a'+index)), EventType: "message",
				ChatID: "oc_review", UserID: test.user,
				Content: "领取 owner/repo PR #42", CreateTimeMs: now.UnixMilli(),
			})
			if err != nil {
				t.Fatalf("Plan: %v", err)
			}
			if test.wantReason != "" {
				if receipt.Accepted || receipt.Reason != test.wantReason {
					t.Fatalf("denied receipt = %#v, want reason %q", receipt, test.wantReason)
				}
				if notice := formatReviewGatewayNotice(receipt.Reason); notice == "" || strings.Contains(notice, "ou_") {
					t.Fatalf("unsafe collaboration denial notice: %q", notice)
				}
				return
			}
			if !receipt.Accepted || receipt.Job == nil || !receipt.Job.CollaborationAuthorized || receipt.Job.CollaborationAdmin != test.wantAdmin {
				t.Fatalf("authorized receipt = %#v", receipt)
			}
		})
	}
}

func TestReviewGatewayClaimExecutionRequiresFrozenAuthorizationAndIdentity(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	job.Action = "claim_review"
	job.RequestedDisplayName = "测试成员"
	identity := ReviewIdentityBinding{
		InstallationID: job.InstallationID, FeishuUserID: job.RequestedBy,
		GitLinkLogin: "gitlink-reviewer", Enabled: true,
	}

	executor := &ReviewGatewayExecutor{Collaboration: store, IdentityBindings: []ReviewIdentityBinding{identity}, Collaborators: &staticReviewCollaboratorReader{collaborators: []ReviewRepositoryCollaborator{{Login: "gitlink-reviewer"}}}}
	if _, err := executor.Execute(context.Background(), job); err == nil || !strings.Contains(err.Error(), "负责人") {
		t.Fatalf("claim without frozen authorization error = %v", err)
	}
	job.CollaborationAuthorized = true
	executor.IdentityBindings = nil
	if _, err := executor.Execute(context.Background(), job); err == nil || !strings.Contains(err.Error(), "尚未绑定 GitLink 身份") {
		t.Fatalf("claim without identity error = %v", err)
	}

	executor.IdentityBindings = []ReviewIdentityBinding{identity}
	result, err := executor.Execute(context.Background(), job)
	if err != nil {
		t.Fatalf("authorized claim: %v", err)
	}
	if result.MutatesGitLink || !result.ReadOnlyGitLink || result.Collaboration == nil || result.Collaboration.GitLinkWrites != 0 {
		t.Fatalf("claim crossed GitLink mutation boundary: %#v", result)
	}
	items, err := store.ListCollaborationItems(context.Background(), job.InstallationID, job.ChatID, job.Repository, "")
	if err != nil || len(items) != 1 || items[0].AssignedTo != job.RequestedBy {
		t.Fatalf("claim state = %#v, err=%v", items, err)
	}
}

func TestReviewClaimConflictIdempotencyAndAdminManagement(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	job.Action = "claim_review"
	job.RequestedDisplayName = "测试成员"
	first, err := store.ApplyCollaborationAction(context.Background(), job, reviewProductInvariantTime.Add(time.Minute))
	if err != nil || first.AssignedTo != job.RequestedBy {
		t.Fatalf("first claim = %#v, err=%v", first, err)
	}
	repeated := job
	repeated.JobID = "job-repeat-own-claim"
	repeated.RequestedDisplayName = ""
	if item, err := store.ApplyCollaborationAction(context.Background(), repeated, reviewProductInvariantTime.Add(2*time.Minute)); err != nil || item.AssignedTo != job.RequestedBy {
		t.Fatalf("idempotent own claim = %#v, err=%v", item, err)
	} else if item.AssignedDisplayName != first.AssignedDisplayName {
		t.Fatalf("idempotent claim downgraded display name: first=%q repeated=%q", first.AssignedDisplayName, item.AssignedDisplayName)
	}

	other := job
	other.JobID = "job-other-claim"
	other.RequestedBy = "ou_other"
	if _, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(3*time.Minute)); err == nil || !strings.Contains(err.Error(), "当前由测试成员负责") || !strings.Contains(err.Error(), "管理员取消领取") || strings.Contains(err.Error(), job.RequestedBy) {
		t.Fatalf("conflicting claim error = %v", err)
	}

	other.Action = "set_review_deadline"
	other.Argument = "2026-08-10"
	if _, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(4*time.Minute)); err == nil {
		t.Fatal("non-owner changed another reviewer's deadline")
	}
	admin := other
	admin.JobID = "job-admin-deadline"
	admin.CollaborationAdmin = true
	deadline, err := store.ApplyCollaborationAction(context.Background(), admin, reviewProductInvariantTime.Add(5*time.Minute))
	if err != nil || deadline.DueAt != "2026-08-10" || deadline.AssignedTo != job.RequestedBy {
		t.Fatalf("admin deadline = %#v, err=%v", deadline, err)
	}

	other.Action = "release_review"
	other.JobID = "job-other-release"
	if _, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(6*time.Minute)); err == nil {
		t.Fatal("non-owner released another reviewer's claim")
	}
	admin = other
	admin.JobID = "job-admin-release"
	admin.CollaborationAdmin = true
	released, err := store.ApplyCollaborationAction(context.Background(), admin, reviewProductInvariantTime.Add(7*time.Minute))
	if err != nil || released.AssignedTo != "" || released.CollaborationStatus != "unassigned" || released.DueAt != "2026-08-10" {
		t.Fatalf("admin release = %#v, err=%v", released, err)
	}

	other.Action = "claim_review"
	other.JobID = "job-other-after-release"
	claimed, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(8*time.Minute))
	if err != nil || claimed.AssignedTo != other.RequestedBy {
		t.Fatalf("claim after release = %#v, err=%v", claimed, err)
	}

	other.Action = "set_review_deadline"
	other.Argument = "2026-08-12"
	other.JobID = "job-deadline-update"
	updated, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(9*time.Minute))
	if err != nil || updated.DueAt != "2026-08-12" || updated.ActionOutcome != "deadline_updated" {
		t.Fatalf("deadline update = %#v, err=%v", updated, err)
	}
	other.Action = "clear_review_deadline"
	other.Argument = ""
	other.JobID = "job-deadline-clear"
	cleared, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(10*time.Minute))
	if err != nil || cleared.DueAt != "" || cleared.ActionOutcome != "deadline_cleared" {
		t.Fatalf("deadline clear = %#v, err=%v", cleared, err)
	}
	other.JobID = "job-deadline-clear-idempotent"
	absent, err := store.ApplyCollaborationAction(context.Background(), other, reviewProductInvariantTime.Add(11*time.Minute))
	if err != nil || absent.ActionOutcome != "deadline_absent" {
		t.Fatalf("idempotent deadline clear = %#v, err=%v", absent, err)
	}
}

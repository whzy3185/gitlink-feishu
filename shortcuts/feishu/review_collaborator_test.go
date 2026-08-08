package feishu

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type staticReviewCollaboratorReader struct {
	collaborators []ReviewRepositoryCollaborator
	err           error
	calls         int
}

func (r *staticReviewCollaboratorReader) ListRepositoryCollaborators(
	context.Context,
	*common.RuntimeContext,
	string,
	string,
) ([]ReviewRepositoryCollaborator, error) {
	r.calls++
	return append([]ReviewRepositoryCollaborator(nil), r.collaborators...), r.err
}

type staticFeishuDisplayNameResolver struct {
	name string
	err  error
}

func (r staticFeishuDisplayNameResolver) ResolveFeishuDisplayName(context.Context, string) (string, error) {
	return r.name, r.err
}

func TestGitLinkReviewRepositoryCollaboratorReaderUsesGETOnlyV1Endpoint(t *testing.T) {
	method, path := "", ""
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		method, path = request.Method, request.URL.Path
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"data":[{"login":"muel","name":"Muel"},{"login":"alice","name":"Alice"}]}`))
	}))
	defer server.Close()
	runtime := &common.RuntimeContext{Client: &client.Client{BaseURL: server.URL + "/api", HTTP: server.Client()}}
	got, err := (GitLinkReviewRepositoryCollaboratorReader{}).ListRepositoryCollaborators(context.Background(), runtime, "owner", "repo")
	if err != nil || len(got) != 2 || got[0].Login != "muel" || got[0].Name != "Muel" {
		t.Fatalf("collaborators = %#v, err=%v", got, err)
	}
	if method != http.MethodGet || path != "/api/v1/owner/repo/collaborators.json" {
		t.Fatalf("request = %s %s", method, path)
	}
}

func TestClaimCollaboratorMembershipAndDisplayNameFallback(t *testing.T) {
	for _, test := range []struct {
		name          string
		collaborators []ReviewRepositoryCollaborator
		readErr       error
		displayName   string
		wantErr       string
		wantDisplay   string
	}{
		{name: "collaborator with Feishu name", collaborators: []ReviewRepositoryCollaborator{{Login: "gitlink-reviewer"}}, displayName: "测试", wantDisplay: "测试"},
		{name: "display lookup falls back to login", collaborators: []ReviewRepositoryCollaborator{{Login: "gitlink-reviewer"}}, wantDisplay: "gitlink-reviewer"},
		{name: "non collaborator", collaborators: []ReviewRepositoryCollaborator{{Login: "someone-else"}}, wantErr: "不是该仓库的协作者"},
		{name: "read failure", readErr: errors.New("gateway timeout"), wantErr: "无法确认当前 GitLink 身份的仓库协作者状态"},
	} {
		t.Run(test.name, func(t *testing.T) {
			store, job := prepareProductInvariantCollaborationStore(t)
			defer store.Close()
			job.Action, job.CollaborationAuthorized = "claim_review", true
			executor := &ReviewGatewayExecutor{
				Collaboration:    store,
				IdentityBindings: []ReviewIdentityBinding{{InstallationID: job.InstallationID, FeishuUserID: job.RequestedBy, GitLinkLogin: "gitlink-reviewer", Enabled: true}},
				Collaborators:    &staticReviewCollaboratorReader{collaborators: test.collaborators, err: test.readErr},
				DisplayNames:     staticFeishuDisplayNameResolver{name: test.displayName, err: map[bool]error{true: errors.New("scope unavailable")}[test.displayName == ""]},
			}
			result, err := executor.Execute(context.Background(), job)
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) || result.Collaboration != nil {
					t.Fatalf("result=%#v err=%v", result, err)
				}
				return
			}
			if err != nil || result.Collaboration == nil || result.Collaboration.Item.AssignedDisplayName != test.wantDisplay {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}
}

func TestGatewayAdminCannotBypassRepositoryCollaboratorMembership(t *testing.T) {
	store, job := prepareProductInvariantCollaborationStore(t)
	defer store.Close()
	job.Action = "claim_review"
	job.CollaborationAuthorized = true
	job.CollaborationAdmin = true
	executor := &ReviewGatewayExecutor{
		Collaboration: store,
		IdentityBindings: []ReviewIdentityBinding{{
			InstallationID: job.InstallationID, FeishuUserID: job.RequestedBy,
			GitLinkLogin: "gateway-admin", Enabled: true,
		}},
		Collaborators: &staticReviewCollaboratorReader{collaborators: []ReviewRepositoryCollaborator{{Login: "repository-collaborator"}}},
	}
	result, err := executor.Execute(context.Background(), job)
	if err == nil || !strings.Contains(err.Error(), "不是该仓库的协作者") || result.Collaboration != nil {
		t.Fatalf("admin bypassed collaborator membership: result=%#v err=%v", result, err)
	}
}

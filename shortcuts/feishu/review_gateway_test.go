package feishu

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larktypes "github.com/larksuite/oapi-sdk-go/v3/channel/types"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

type reviewGatewayMockHTTPClient struct {
	do func(*http.Request) (*http.Response, error)
}

func (c reviewGatewayMockHTTPClient) Do(request *http.Request) (*http.Response, error) {
	return c.do(request)
}

func TestParseReviewGatewayIntent(t *testing.T) {
	tests := []struct {
		input      string
		wantName   string
		wantRepo   string
		wantNumber int
	}{
		{input: "帮助", wantName: "help"},
		{input: "查看绑定", wantName: "show_binding"},
		{input: "查看待 Review", wantName: "read_review_queue"},
		{input: "查看 owner/repo 待 Review", wantName: "read_review_queue", wantRepo: "owner/repo"},
		{input: "查看 PR #431", wantName: "read_review_context", wantNumber: 431},
		{input: "查看 owner/repo PR #431", wantName: "read_review_context", wantRepo: "owner/repo", wantNumber: 431},
		{input: "仓库列表", wantName: "list_repositories"},
		{input: "领取 PR 431", wantName: "claim_review", wantNumber: 431},
		{input: "释放 PR #431", wantName: "release_review", wantNumber: 431},
		{input: "准备提交 PR #431 Review", wantName: "prepare_common_review", wantNumber: 431},
		{input: "生成 PR #431 Review 草稿", wantName: "generate_review_draft", wantNumber: 431},
		{input: "刷新 PR #431", wantName: "refresh_review_context", wantNumber: 431},
		{input: "绑定仓库 gitlink-org/gitlink-cli", wantName: "plan_bind_repository", wantRepo: "gitlink-org/gitlink-cli"},
		{input: "批准 PR #431", wantName: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			got := parseReviewGatewayIntent(test.input)
			if got.Name != test.wantName || got.Repository != test.wantRepo || got.PRNumber != test.wantNumber {
				t.Fatalf("parseReviewGatewayIntent(%q) = %#v", test.input, got)
			}
		})
	}
}

func TestParseReviewGatewayControlledWriteIntents(t *testing.T) {
	deadline := parseReviewGatewayIntent("设置 PR #431 截止 2026-08-02")
	if deadline.Name != "set_review_deadline" || deadline.PRNumber != 431 || deadline.Argument != "2026-08-02" {
		t.Fatalf("deadline intent = %#v", deadline)
	}
	confirmation := parseReviewGatewayIntent("确认 Review review-plan-1234abcd")
	if confirmation.Name != "confirm_common_review" || confirmation.Argument != "review-plan-1234abcd" {
		t.Fatalf("confirmation intent = %#v", confirmation)
	}
}

func TestReviewGatewayPlansBoundReadOnlyJobAndDeduplicatesMessage(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingV1,
		Bindings: []ReviewChatBinding{{
			ChatID:         "oc_review",
			Repository:     "gitlink-org/gitlink-cli",
			Enabled:        true,
			AllowedUserIDs: []string{"ou_owner"},
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	event := ReviewGatewayEvent{
		EventID:      "evt_1",
		MessageID:    "om_1",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看 gitlink-org/gitlink-cli PR #431",
		CreateTimeMs: now.Add(-time.Minute).UnixMilli(),
	}
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !receipt.Accepted || receipt.Job == nil {
		t.Fatalf("receipt not accepted: %#v", receipt)
	}
	if receipt.Job.MutatesGitLink || receipt.Job.Mode != "preview" || receipt.Job.Repository != "gitlink-org/gitlink-cli" {
		t.Fatalf("unsafe or incomplete job: %#v", receipt.Job)
	}
	if receipt.DedupeKey != "feishu:message:om_1" {
		t.Fatalf("dedupe key = %q", receipt.DedupeKey)
	}

	event.EventID = "evt_retry"
	duplicate, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("duplicate Plan: %v", err)
	}
	if duplicate.Accepted || !duplicate.Duplicate || duplicate.Reason != "duplicate_event" {
		t.Fatalf("duplicate receipt = %#v", duplicate)
	}
}

func TestReviewGatewayRejectsUnsupportedBindingSchema(t *testing.T) {
	_, err := NewReviewGateway(
		ReviewGatewayBindings{SchemaVersion: "feishu.review-bindings/v3"},
		ReviewGatewayConfig{},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), reviewGatewayBindingSchema) {
		t.Fatalf("unsupported binding schema error = %v", err)
	}
}

func TestReviewGatewayUpgradesV1BindingAndResolvesV2MultiRepository(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	legacy, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingV1,
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_legacy",
			Repository: "owner/legacy",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("upgrade v1 binding: %v", err)
	}
	legacyReceipt, err := legacy.Plan(ReviewGatewayEvent{
		MessageID: "om_legacy", EventType: "message", ChatID: "oc_legacy",
		UserID: "ou_owner", Content: "查看 owner/legacy PR #1", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !legacyReceipt.Accepted || legacyReceipt.Job.Repository != "owner/legacy" {
		t.Fatalf("legacy receipt = %#v, err=%v", legacyReceipt, err)
	}

	multi, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "main",
			GitLinkHost:         "https://www.gitlink.org.cn",
			OperationMode:       "collaborate",
			AllowedRepositories: []string{"owner/one", "owner/two"},
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID:            "oc_multi",
			InstallationID:    "main",
			Repositories:      []string{"owner/one", "owner/two"},
			DefaultRepository: "owner/one",
			Enabled:           true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway multi: %v", err)
	}
	defaultReceipt, err := multi.Plan(ReviewGatewayEvent{
		MessageID: "om_default", EventType: "message", ChatID: "oc_multi",
		UserID: "ou_owner", Content: "查看 PR #12", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || defaultReceipt.Accepted || defaultReceipt.Reason != "repository_qualification_required" {
		t.Fatalf("default multi receipt = %#v, err=%v", defaultReceipt, err)
	}
	explicitReceipt, err := multi.Plan(ReviewGatewayEvent{
		MessageID: "om_explicit", EventType: "message", ChatID: "oc_multi",
		UserID: "ou_owner", Content: "查看 owner/two PR #12", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !explicitReceipt.Accepted || explicitReceipt.Job.Repository != "owner/two" ||
		len(explicitReceipt.Job.Repositories) != 2 {
		t.Fatalf("explicit multi receipt = %#v, err=%v", explicitReceipt, err)
	}
	outOfScope, err := multi.Plan(ReviewGatewayEvent{
		MessageID: "om_scope", EventType: "message", ChatID: "oc_multi",
		UserID: "ou_owner", Content: "查看 other/repo PR #12", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || outOfScope.Accepted || outOfScope.Reason != "repository_not_bound" {
		t.Fatalf("out-of-scope receipt = %#v, err=%v", outOfScope, err)
	}
}

func TestReviewGatewayPublicReadDiscoveryFollowsGitHubAppBoundary(t *testing.T) {
	now := time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "main",
			GitLinkHost:         "https://www.gitlink.org.cn",
			CredentialRef:       "env:GITLINK_INSTALLATION_TOKEN",
			OperationMode:       "write",
			AllowedRepositories: []string{"owner/bound"},
			AllowPublicRead:     true,
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID:            "oc_public",
			InstallationID:    "main",
			Repositories:      []string{"owner/bound"},
			DefaultRepository: "owner/bound",
			AllowPublicRead:   true,
			Enabled:           true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}

	publicRead, err := gateway.Plan(ReviewGatewayEvent{
		MessageID: "om_public", EventType: "message", ChatID: "oc_public",
		UserID: "ou_reviewer", Content: "查看 public/repo PR #9", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !publicRead.Accepted || publicRead.Job == nil ||
		!publicRead.Job.PublicRead || publicRead.Job.MutatesGitLink ||
		publicRead.Job.Repository != "public/repo" {
		t.Fatalf("public read receipt = %#v, err=%v", publicRead, err)
	}

	boundRead, err := gateway.Plan(ReviewGatewayEvent{
		MessageID: "om_bound", EventType: "message", ChatID: "oc_public",
		UserID: "ou_reviewer", Content: "查看 owner/bound PR #9", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !boundRead.Accepted || boundRead.Job == nil || boundRead.Job.PublicRead {
		t.Fatalf("bound read receipt = %#v, err=%v", boundRead, err)
	}

	for index, content := range []string{
		"生成 public/repo PR #9 Review 草稿",
		"启动 public/repo PR #9 Agent 审查",
		"领取 public/repo PR #9",
		"准备提交 public/repo PR #9 Review",
	} {
		receipt, planErr := gateway.Plan(ReviewGatewayEvent{
			MessageID: fmt.Sprintf("om_denied_%d", index), EventType: "message", ChatID: "oc_public",
			UserID: "ou_reviewer", Content: content, CreateTimeMs: now.UnixMilli(),
		})
		if planErr != nil || receipt.Accepted || receipt.Reason != "repository_not_bound" {
			t.Fatalf("public non-read command %q = %#v, err=%v", content, receipt, planErr)
		}
	}
}

func TestReviewGatewayPublicReadRequiresInstallationAndChatOptIn(t *testing.T) {
	_, err := normalizeReviewGatewayBindings(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID: "main", OperationMode: "observe",
			AllowedRepositories: []string{"owner/repo"}, Enabled: true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID: "oc_public", InstallationID: "main",
			Repositories: []string{"owner/repo"}, AllowPublicRead: true, Enabled: true,
		}},
	})
	if err == nil || !strings.Contains(err.Error(), "disables it") {
		t.Fatalf("mismatched public read policy error = %v", err)
	}
}

func TestReviewGatewayV2DocumentationExampleIsValid(t *testing.T) {
	payload, err := os.ReadFile("../../docs/examples/feishu-review-bindings-v2.json")
	if err != nil {
		t.Fatalf("read v2 example: %v", err)
	}
	var bindings ReviewGatewayBindings
	if err := json.Unmarshal(payload, &bindings); err != nil {
		t.Fatalf("decode v2 example: %v", err)
	}
	normalized, err := normalizeReviewGatewayBindings(bindings)
	if err != nil {
		t.Fatalf("validate v2 example: %v", err)
	}
	if normalized.SchemaVersion != reviewGatewayBindingSchema ||
		len(normalized.Installations) != 1 ||
		len(normalized.Bindings) != 1 ||
		len(normalized.Bindings[0].Repositories) != 2 {
		t.Fatalf("normalized example = %#v", normalized)
	}
}

func TestReviewGatewayRequiresExplicitRepositoryForEveryScopedAction(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "main",
			OperationMode:       "observe",
			AllowedRepositories: []string{"owner/one", "owner/two"},
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID: "oc_multi", InstallationID: "main",
			Repositories: []string{"owner/one", "owner/two"}, Enabled: true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	for index, command := range []string{
		"查看待审查",
		"查看 PR #12",
		"刷新 PR #12",
		"生成 PR #12 Review 草稿",
		"启动 PR #12 Agent 审查",
		"领取 PR #12",
		"释放 PR #12",
		"设置 PR #12 截止 2026-08-05",
		"准备提交 PR #12 Review",
	} {
		receipt, planErr := gateway.Plan(ReviewGatewayEvent{
			MessageID: fmt.Sprintf("om_unqualified_%d", index), EventType: "message", ChatID: "oc_multi",
			UserID: "ou_owner", Content: command, CreateTimeMs: now.UnixMilli(),
		})
		if planErr != nil || receipt.Accepted || receipt.Job != nil || receipt.Reason != "repository_qualification_required" {
			t.Fatalf("unqualified command %q = %#v, err=%v", command, receipt, planErr)
		}
	}
	qualifiedQueue, err := gateway.Plan(ReviewGatewayEvent{
		MessageID: "om_qualified_queue", EventType: "message", ChatID: "oc_multi",
		UserID: "ou_owner", Content: "查看 owner/two 待审查", CreateTimeMs: now.UnixMilli(),
	})
	if err != nil || !qualifiedQueue.Accepted || qualifiedQueue.Job == nil || qualifiedQueue.Job.Repository != "owner/two" {
		t.Fatalf("qualified queue receipt = %#v, err=%v", qualifiedQueue, err)
	}
}

func TestReviewGatewayInstallationModeGatesControlledWrite(t *testing.T) {
	now := time.Date(2026, 8, 1, 9, 0, 0, 0, time.UTC)
	newGateway := func(mode string, enabled bool) *ReviewGateway {
		t.Helper()
		gateway, err := NewReviewGateway(ReviewGatewayBindings{
			SchemaVersion: reviewGatewayBindingSchema,
			Installations: []GitLinkInstallation{{
				InstallationID: "main", OperationMode: mode,
				CredentialRef:       "env:GITLINK_TOKEN",
				AllowedRepositories: []string{"owner/repo"}, Enabled: enabled,
			}},
			Bindings: []ReviewChatBinding{{
				ChatID: "oc_review", InstallationID: "main",
				Repositories: []string{"owner/repo"}, Enabled: true,
			}},
		}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
		if err != nil {
			t.Fatalf("NewReviewGateway: %v", err)
		}
		return gateway
	}
	for name, test := range map[string]struct {
		gateway *ReviewGateway
		reason  string
	}{
		"collaborate": {gateway: newGateway("collaborate", true), reason: "installation_write_disabled"},
		"disabled":    {gateway: newGateway("write", false), reason: "installation_disabled"},
	} {
		t.Run(name, func(t *testing.T) {
			receipt, err := test.gateway.Plan(ReviewGatewayEvent{
				MessageID: "om_" + name, EventType: "message", ChatID: "oc_review",
				UserID: "ou_owner", Content: "确认 Review review-plan-1234abcd", CreateTimeMs: now.UnixMilli(),
			})
			if err != nil || receipt.Accepted || receipt.Reason != test.reason {
				t.Fatalf("receipt = %#v, err=%v", receipt, err)
			}
		})
	}
}

func TestReviewGatewayRuntimeUsesInstallationCredentialInsteadOfGlobalToken(t *testing.T) {
	t.Setenv("GITLINK_TOKEN", "global-token-must-not-leak")
	t.Setenv("INSTALLATION_TOKEN", "installation-token")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Query().Get("access_token") != "installation-token" {
			t.Fatalf("access token = %q", request.URL.Query().Get("access_token"))
		}
		if request.URL.Path != "/api/test.json" {
			t.Fatalf("request path = %q", request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":0,"data":{"ok":true}}`))
	}))
	defer server.Close()
	baseRuntime := &common.RuntimeContext{
		Client: &client.Client{BaseURL: "https://unused.example/api", HTTP: server.Client()},
		Args:   map[string]string{},
	}
	executor := &ReviewGatewayExecutor{
		Runtime: baseRuntime,
		Installations: map[string]GitLinkInstallation{
			"main": {
				InstallationID: "main", GitLinkHost: server.URL,
				CredentialRef: "env:INSTALLATION_TOKEN", OperationMode: "write", Enabled: true,
			},
		},
		RequireInstallationRuntime: true,
	}
	runtime, err := executor.runtimeForJob(ReviewGatewayJob{InstallationID: "main"})
	if err != nil {
		t.Fatalf("runtimeForJob: %v", err)
	}
	if _, err := runtime.CallAPI(http.MethodGet, "/test", nil); err != nil {
		t.Fatalf("installation CallAPI: %v", err)
	}
	if baseRuntime.Client.BaseURL != "https://unused.example/api" {
		t.Fatalf("base runtime was mutated: %q", baseRuntime.Client.BaseURL)
	}
}

func TestReviewGatewayPublicReadRuntimeStripsInstallationCredential(t *testing.T) {
	t.Setenv("INSTALLATION_TOKEN", "must-not-leak")
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if token := request.URL.Query().Get("access_token"); token != "" {
			t.Fatalf("public read leaked access token %q", token)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"status":0,"data":{"ok":true}}`))
	}))
	defer server.Close()
	executor := &ReviewGatewayExecutor{
		Runtime: &common.RuntimeContext{
			Client: &client.Client{BaseURL: server.URL, HTTP: server.Client()},
			Args:   map[string]string{},
		},
		Installations: map[string]GitLinkInstallation{
			"main": {
				InstallationID: "main", GitLinkHost: server.URL,
				CredentialRef: "env:INSTALLATION_TOKEN", OperationMode: "write", Enabled: true,
			},
		},
		RequireInstallationRuntime: true,
	}
	runtime, err := executor.runtimeForJob(ReviewGatewayJob{
		InstallationID: "main", Action: "read_review_context", PublicRead: true,
	})
	if err != nil {
		t.Fatalf("runtimeForJob: %v", err)
	}
	if _, err := runtime.CallAPI(http.MethodGet, "/test", nil); err != nil {
		t.Fatalf("public read CallAPI: %v", err)
	}
	if _, err := executor.runtimeForJob(ReviewGatewayJob{
		InstallationID: "main", Action: "generate_review_draft", PublicRead: true,
	}); err == nil {
		t.Fatal("public read runtime accepted a non-read action")
	}
}

func TestReviewGatewayPublicReadRequiresVerifiedPublicRepository(t *testing.T) {
	if reviewContextRepositoryIsPublic(workflow.ReviewContext{}) {
		t.Fatal("missing repository visibility was treated as public")
	}
	if reviewContextRepositoryIsPublic(workflow.ReviewContext{
		RepositoryInfo: map[string]interface{}{"is_public": false},
	}) {
		t.Fatal("private repository was treated as public")
	}
	if !reviewContextRepositoryIsPublic(workflow.ReviewContext{
		RepositoryInfo: map[string]interface{}{"is_public": true},
	}) {
		t.Fatal("verified public repository was rejected")
	}
}

func TestReviewGatewayPublicReadDoesNotCreateCollaborationResources(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/api/v1/public/repo.json":
			_, _ = writer.Write([]byte(`{"is_public":true,"full_name":"public/repo"}`))
		case "/api/public/repo/pulls/9.json":
			_, _ = writer.Write([]byte(`{"pull_request":{"number":9,"title":"Public PR","state":"open","head_commit_sha":"head-public"}}`))
		case "/api/v1/public/repo/pulls/9/versions.json":
			_, _ = writer.Write([]byte(`{"versions":[{"id":1,"head_commit_sha":"head-public","files_count":1,"commits_count":1}]}`))
		case "/api/public/repo/pulls/9/files.json":
			_, _ = writer.Write([]byte(`{"files":[{"filename":"README.md","additions":1}]}`))
		case "/api/v1/public/repo/pulls/9/reviews.json":
			_, _ = writer.Write([]byte(`{"reviews":[]}`))
		case "/api/v1/public/repo/pulls/9/journals.json":
			_, _ = writer.Write([]byte(`{"journals":[]}`))
		default:
			t.Fatalf("unexpected public read request: %s", request.URL.Path)
		}
	}))
	defer server.Close()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "public-read.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	publisher := &recordingReviewCollaborationPublisher{}
	executor := &ReviewGatewayExecutor{
		Runtime: &common.RuntimeContext{
			Client: &client.Client{BaseURL: server.URL + "/api", HTTP: server.Client()},
			Args:   map[string]string{},
		},
		Installations: map[string]GitLinkInstallation{
			"main": {
				InstallationID: "main", GitLinkHost: server.URL + "/api",
				OperationMode: "collaborate", Enabled: true,
			},
		},
		RequireInstallationRuntime: true,
		Collaboration:              store,
		Publisher:                  publisher,
	}
	result, err := executor.Execute(context.Background(), ReviewGatewayJob{
		JobID: "job-public", Action: "read_review_context", Mode: "preview",
		InstallationID: "main", Repository: "public/repo", PRNumber: 9,
		RequestedBy: "ou_reader", PublicRead: true,
	})
	if err != nil || result.Status != "completed" || !result.PublicRead || result.MutatesGitLink {
		t.Fatalf("public read result = %#v, err=%v", result, err)
	}
	if publisher.calls != 0 || len(result.ResourceSync) != 0 || result.Collaboration != nil {
		t.Fatalf("public read created collaboration projection: %#v, calls=%d", result, publisher.calls)
	}
	cardJSON, cardOK := safeReviewGatewayCardJSON(result.ResultCard)
	if !cardOK || !strings.Contains(cardJSON, "Public PR") ||
		strings.Contains(cardJSON, "Base / Doc / Task") || strings.Contains(cardJSON, "负责人") {
		t.Fatalf("public read card boundary = %s, ok=%t", cardJSON, cardOK)
	}
	var collaborationItems int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM review_collaboration_items").Scan(&collaborationItems); err != nil {
		t.Fatalf("count collaboration items: %v", err)
	}
	if collaborationItems != 0 {
		t.Fatalf("public read persisted %d collaboration items", collaborationItems)
	}
}

func TestReviewIdentityBindingsAreScopedPerInstallation(t *testing.T) {
	bindings, err := normalizeReviewGatewayBindings(ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{
			{InstallationID: "one", OperationMode: "observe", AllowedRepositories: []string{"owner/one"}, Enabled: true},
			{InstallationID: "two", OperationMode: "observe", AllowedRepositories: []string{"owner/two"}, Enabled: true},
		},
		IdentityBindings: []ReviewIdentityBinding{
			{InstallationID: "one", FeishuUserID: "ou_same", GitLinkLogin: "alice-one", Enabled: true},
			{InstallationID: "two", FeishuUserID: "ou_same", GitLinkLogin: "alice-two", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("normalizeReviewGatewayBindings: %v", err)
	}
	one, ok := findReviewIdentity(bindings.IdentityBindings, "ou_same", "one")
	if !ok || one.GitLinkLogin != "alice-one" {
		t.Fatalf("installation one identity = %#v, ok=%t", one, ok)
	}
	two, ok := findReviewIdentity(bindings.IdentityBindings, "ou_same", "two")
	if !ok || two.GitLinkLogin != "alice-two" {
		t.Fatalf("installation two identity = %#v, ok=%t", two, ok)
	}
	if _, ok := findReviewIdentity(bindings.IdentityBindings, "ou_same", ""); ok {
		t.Fatal("unscoped lookup accepted an ambiguous cross-installation identity")
	}
}

func TestReviewGatewayRejectsStaleUnauthorizedAndUnsupportedEvents(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:         "oc_review",
			Repository:     "gitlink-org/gitlink-cli",
			Enabled:        true,
			AllowedUserIDs: []string{"ou_owner"},
		}},
	}, ReviewGatewayConfig{
		Now:         func() time.Time { return now },
		StaleWindow: 10 * time.Minute,
	}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	base := ReviewGatewayEvent{
		EventID:      "evt",
		MessageID:    "om",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看 owner/repo 待审查",
		CreateTimeMs: now.UnixMilli(),
	}

	stale := base
	stale.MessageID = "om_stale"
	stale.CreateTimeMs = now.Add(-11 * time.Minute).UnixMilli()
	assertReviewGatewayReason(t, gateway, stale, "stale_event")

	unauthorized := base
	unauthorized.MessageID = "om_unauthorized"
	unauthorized.UserID = "ou_other"
	assertReviewGatewayReason(t, gateway, unauthorized, "sender_not_allowed")

	unsupported := base
	unsupported.MessageID = "om_unsupported"
	unsupported.EventType = "url_verification"
	assertReviewGatewayReason(t, gateway, unsupported, "unsupported_event_type")
}

func TestReviewGatewayBindingPlanRequiresAdminAndDoesNotApply(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{}, ReviewGatewayConfig{
		AdminUserIDs: []string{"ou_admin"},
		Now:          func() time.Time { return now },
	}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	event := ReviewGatewayEvent{
		EventID:      "evt_bind",
		EventType:    "card_action",
		ChatID:       "oc_unbound",
		UserID:       "ou_admin",
		Content:      "绑定仓库 owner/repo",
		CreateTimeMs: now.UnixMilli(),
	}
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !receipt.Accepted || receipt.Bound || receipt.Job == nil || !receipt.Job.CollaborationMutation {
		t.Fatalf("binding plan receipt = %#v", receipt)
	}
	show := event
	show.EventID = "evt_show"
	show.Content = "查看绑定"
	assertReviewGatewayReason(t, gateway, show, "chat_not_bound")
}

func TestSQLiteReviewGatewayStorePersistsDedupeAndRedactsErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway.db")
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	reserved, err := store.Reserve("feishu:message:om_1", now, 24*time.Hour)
	if err != nil || !reserved {
		t.Fatalf("Reserve = %v, %v", reserved, err)
	}
	job := ReviewGatewayJob{
		SchemaVersion:  reviewGatewayJobSchema,
		JobID:          "job-test",
		DedupeKey:      "feishu:message:om_1",
		Status:         "queued",
		Action:         "read_review_context",
		Repository:     "owner/repo",
		PRNumber:       42,
		ChatID:         "oc_review",
		RequestedBy:    "ou_owner",
		CreatedAt:      now.Format(time.RFC3339),
		Mode:           "preview",
		MutatesGitLink: false,
	}
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	rawError := "GET https://user:pass@example.test/api?access_token=url-secret Authorization: Bearer bearer-secret Cookie: session=cookie-secret"
	if err := store.UpdateJobStatus(context.Background(), job.JobID, "failed", rawError); err != nil {
		t.Fatalf("UpdateJobStatus: %v", err)
	}
	var errorSummary string
	if err := store.db.QueryRow("SELECT error_summary FROM review_gateway_jobs WHERE job_id = ?", job.JobID).Scan(&errorSummary); err != nil {
		t.Fatalf("read error summary: %v", err)
	}
	for _, secret := range []string{"url-secret", "bearer-secret", "cookie-secret", "user:pass"} {
		if strings.Contains(errorSummary, secret) {
			t.Fatalf("error summary leaked %q: %s", secret, errorSummary)
		}
	}
	if !strings.Contains(errorSummary, "***") {
		t.Fatalf("error summary missing redaction marker: %s", errorSummary)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen store: %v", err)
	}
	defer reopened.Close()
	reserved, err = reopened.Reserve("feishu:message:om_1", now.Add(time.Minute), 24*time.Hour)
	if err != nil {
		t.Fatalf("Reserve after restart: %v", err)
	}
	if reserved {
		t.Fatal("dedupe reservation was lost across SQLite restart")
	}
}

func TestSQLiteReviewGatewayStoreSyncsValidatedInstallationConfiguration(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "installations.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 1, 10, 0, 0, 0, time.UTC)
	bindings := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "production",
			GitLinkHost:         "https://www.gitlink.org.cn/",
			Owner:               "owner",
			CredentialRef:       "env:GITLINK_REVIEW_TOKEN",
			OperationMode:       "write",
			AllowedRepositories: []string{"owner/one", "owner/two"},
			AllowPublicRead:     true,
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID:            "oc_review",
			InstallationID:    "production",
			Repositories:      []string{"owner/two", "owner/one"},
			DefaultRepository: "owner/one",
			AllowPublicRead:   true,
			Enabled:           true,
			AdminUserIDs:      []string{"ou_admin"},
			AllowedUserIDs:    []string{"ou_owner"},
		}},
		IdentityBindings: []ReviewIdentityBinding{{
			InstallationID: "production", FeishuUserID: "ou_owner",
			GitLinkLogin: "gitlink-owner", VerificationMethod: "admin_config", Enabled: true,
		}},
	}
	if err := store.SyncReviewGatewayConfiguration(context.Background(), bindings, "test", now); err != nil {
		t.Fatalf("SyncReviewGatewayConfiguration: %v", err)
	}
	var installationCount, repositoryCount, bindingCount, identityCount, auditCount int
	for query, target := range map[string]*int{
		"SELECT COUNT(*) FROM gitlink_installations":              &installationCount,
		"SELECT COUNT(*) FROM installation_repositories":          &repositoryCount,
		"SELECT COUNT(*) FROM chat_repository_bindings":           &bindingCount,
		"SELECT COUNT(*) FROM review_identity_bindings":           &identityCount,
		"SELECT COUNT(*) FROM review_gateway_configuration_audit": &auditCount,
	} {
		if err := store.db.QueryRow(query).Scan(target); err != nil {
			t.Fatalf("query configuration count %q: %v", query, err)
		}
	}
	if installationCount != 1 || repositoryCount != 2 || bindingCount != 2 || identityCount != 1 || auditCount != 1 {
		t.Fatalf("configuration counts = installation:%d repository:%d binding:%d identity:%d audit:%d",
			installationCount, repositoryCount, bindingCount, identityCount, auditCount)
	}
	var credentialRef, defaultRepository, adminsJSON string
	var installationPublicRead, bindingPublicRead int
	if err := store.db.QueryRow(
		"SELECT credential_ref, allow_public_read FROM gitlink_installations WHERE installation_id = ?",
		"production",
	).Scan(&credentialRef, &installationPublicRead); err != nil {
		t.Fatalf("read credential reference: %v", err)
	}
	if err := store.db.QueryRow(
		"SELECT repository, admin_user_ids_json, allow_public_read FROM chat_repository_bindings WHERE is_default = 1",
	).Scan(&defaultRepository, &adminsJSON, &bindingPublicRead); err != nil {
		t.Fatalf("read default binding: %v", err)
	}
	if credentialRef != "env:GITLINK_REVIEW_TOKEN" || defaultRepository != "owner/one" || adminsJSON != `["ou_admin"]` ||
		installationPublicRead != 1 || bindingPublicRead != 1 {
		t.Fatalf("persisted configuration = credential:%q default:%q admins:%q", credentialRef, defaultRepository, adminsJSON)
	}

	invalid := bindings
	invalid.Installations[0].AllowedRepositories = []string{"*"}
	if err := store.SyncReviewGatewayConfiguration(context.Background(), invalid, "test", now.Add(time.Minute)); err == nil {
		t.Fatal("invalid wildcard configuration was accepted")
	}
	if err := store.db.QueryRow("SELECT COUNT(*) FROM installation_repositories").Scan(&repositoryCount); err != nil {
		t.Fatalf("read repository count after rejected sync: %v", err)
	}
	if repositoryCount != 2 {
		t.Fatalf("rejected sync changed committed configuration: repository count %d", repositoryCount)
	}
}

func TestReviewGatewayQueueRunsAsynchronously(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{Now: func() time.Time { return now }}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	store := NewMemoryReviewGatewayJobStore()
	completed := make(chan ReviewGatewayJobOutcome, 1)
	queue := NewReviewGatewayQueue(gateway, store, 1, func(outcome ReviewGatewayJobOutcome) {
		completed <- outcome
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go queue.Run(ctx, func(_ context.Context, job ReviewGatewayJob) (ReviewGatewayExecutionResult, error) {
		return ReviewGatewayExecutionResult{
			SchemaVersion: reviewGatewayResultSchema,
			JobID:         job.JobID,
			Status:        "completed",
			Mode:          "preview",
		}, nil
	})

	receipt := queue.Enqueue(ctx, ReviewGatewayEvent{
		EventID:      "evt_queue",
		MessageID:    "om_queue",
		EventType:    "message",
		ChatID:       "oc_review",
		UserID:       "ou_owner",
		Content:      "查看 owner/repo 待审查",
		CreateTimeMs: now.UnixMilli(),
	})
	if !receipt.Accepted || receipt.Job == nil {
		t.Fatalf("queue receipt = %#v", receipt)
	}
	select {
	case outcome := <-completed:
		if outcome.Err != nil {
			t.Fatalf("queue handler: %v", outcome.Err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for asynchronous job")
	}
	store.mu.Lock()
	status := store.Status[receipt.Job.JobID]
	store.mu.Unlock()
	if status != "completed" {
		t.Fatalf("job status = %q", status)
	}
}

func TestPlanReviewSnapshotSyncProtectsPartialAndManualState(t *testing.T) {
	base := workflow.ReviewContext{
		CollectionStatus: "complete",
		CurrentHeadSHA:   "head-2",
		WorkItem: workflow.ReviewWorkItem{
			GitLinkState:      "open",
			SourceFingerprint: "fingerprint-2",
			SourceScope:       workflow.ReviewSourceScope{Complete: true},
		},
	}
	if got := PlanReviewSnapshotSync(base, nil); got.Action != "apply" || !got.PreserveManualFields || !got.OverwriteGitLinkFacts {
		t.Fatalf("complete changed snapshot plan = %#v", got)
	}
	partial := base
	partial.Partial = true
	partial.CollectionStatus = "partial"
	if got := PlanReviewSnapshotSync(partial, nil); got.Action != "preserve_previous" || got.OverwriteGitLinkFacts {
		t.Fatalf("partial snapshot plan = %#v", got)
	}
	merged := base
	merged.WorkItem.GitLinkState = "merged"
	if got := PlanReviewSnapshotSync(merged, nil); got.Action != "archive" || !got.Archive {
		t.Fatalf("merged snapshot plan = %#v", got)
	}
	previous := &ReviewSnapshotState{
		SourceFingerprint: "fingerprint-2",
		HeadSHA:           "head-2",
		Complete:          true,
	}
	if got := PlanReviewSnapshotSync(base, previous); got.Action != "unchanged" || !got.PreserveManualFields {
		t.Fatalf("unchanged snapshot plan = %#v", got)
	}
}

func TestReviewGatewaySDKEventNormalizationExcludesCardToken(t *testing.T) {
	message := reviewGatewayEventFromMessage(&larktypes.NormalizedMessage{
		EventID:      "evt_message",
		MessageID:    "om_message",
		ChatID:       "oc_review",
		ChatType:     "group",
		UserID:       "ou_owner",
		Content:      "查看 owner/repo PR #42",
		CreateTimeMs: 123,
	})
	if message.EventType != "message" || message.MessageID != "om_message" || message.Content != "查看 owner/repo PR #42" {
		t.Fatalf("normalized message = %#v", message)
	}

	mentioned := reviewGatewayEventFromMessage(&larktypes.NormalizedMessage{
		EventID:   "evt_mentioned",
		MessageID: "om_mentioned",
		ChatID:    "oc_review",
		ChatType:  "group",
		UserID:    "ou_owner",
		Content:   "@_user_1  查看 owner/repo PR #42",
		Mentions: []larktypes.Mention{
			{Key: "@_user_1", OpenID: "ou_bot", IsBot: true},
		},
	})
	if mentioned.Content != "查看 owner/repo PR #42" {
		t.Fatalf("bot mention was not removed from command: %#v", mentioned)
	}

	card, ok := reviewGatewayEventFromCardAction(&larktypes.CardActionEvent{
		EventID:   "evt_card",
		MessageID: "om_card",
		ChatID:    "oc_review",
		Token:     "must-not-be-copied",
		Operator:  larktypes.CardActionOperator{OpenID: "ou_owner"},
		Action: larktypes.CardActionPayload{
			Value: map[string]interface{}{"command": "刷新 owner/repo PR #42"},
		},
	})
	if !ok || card.EventType != "card_action" || card.Content != "刷新 owner/repo PR #42" {
		t.Fatalf("normalized card action = %#v, %v", card, ok)
	}
	encoded, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	if strings.Contains(string(encoded), "must-not-be-copied") {
		t.Fatalf("card callback token leaked into gateway event: %s", encoded)
	}
}

func TestReviewDraftPreviewUsesStableBoundedTemplate(t *testing.T) {
	reviewContext := workflow.ReviewContext{
		Repository:       "owner/repo",
		PullRequest:      42,
		CurrentVersionID: "7",
		CurrentPatchset:  workflow.ReviewContextPatchset{FilesCount: 3},
		Summary: workflow.ReviewCollaborationSummary{
			Decision:     "changes_pending",
			TotalReviews: 2,
			OpenThreads:  1,
		},
		ReviewerSummaries: []workflow.ReviewContextReviewerSummary{{
			ReviewerKey:     "alice",
			Actor:           "Alice",
			CurrentDecision: "approved",
		}},
		Threads: []workflow.ReviewContextThread{{
			Author:    "Bob",
			Content:   strings.Repeat("需", 450),
			State:     "open",
			Freshness: "current",
			Path:      "main.go",
			LineCode:  "12",
		}},
		WorkItem: workflow.ReviewWorkItem{
			Title:               "Improve review flow",
			Unknowns:            []string{"CI state unavailable"},
			RecommendedNextStep: "owner decision",
		},
	}
	draft := buildReviewDraftPreview(reviewContext)
	if draft.TemplateVersion != "review-draft/v1" || len(draft.ReviewerStates) != 1 || len(draft.OpenThreads) != 1 {
		t.Fatalf("draft = %#v", draft)
	}
	if len([]rune(draft.OpenThreads[0].Content)) != 401 {
		t.Fatalf("thread preview length = %d", len([]rune(draft.OpenThreads[0].Content)))
	}
}

func assertReviewGatewayReason(t *testing.T, gateway *ReviewGateway, event ReviewGatewayEvent, want string) {
	t.Helper()
	receipt, err := gateway.Plan(event)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if receipt.Accepted || receipt.Reason != want {
		t.Fatalf("receipt = %#v, want reason %q", receipt, want)
	}
}

func TestSQLiteReviewGatewayStoreRecoversExpiredRunningJob(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway-recovery.db")
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	job := testReviewGatewayJob(now, "job-recovery")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-1",
		Now:           now,
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(claimed) != 1 || claimed[0].AttemptCount != 1 {
		t.Fatalf("first claim = %#v, %v", claimed, err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	reopened, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()
	notExpired, err := reopened.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-2",
		Now:           now.Add(30 * time.Second),
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(notExpired) != 0 {
		t.Fatalf("unexpired lease was recovered: %#v, %v", notExpired, err)
	}
	recovered, err := reopened.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner:    "worker-2",
		Now:           now.Add(2 * time.Minute),
		LeaseDuration: time.Minute,
		Limit:         1,
	})
	if err != nil || len(recovered) != 1 {
		t.Fatalf("expired job recovery = %#v, %v", recovered, err)
	}
	if recovered[0].AttemptCount != 2 || recovered[0].LeaseOwner != "worker-2" {
		t.Fatalf("recovered job metadata = %#v", recovered[0])
	}
}

func TestSQLiteReviewGatewayQueueAtomicallyDeduplicatesAndRecoversOrphanReservation(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-atomic.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	gateway, err := NewReviewGateway(
		ReviewGatewayBindings{Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}}},
		ReviewGatewayConfig{Now: func() time.Time { return now }},
		store,
	)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 2, nil)
	queue.now = func() time.Time { return now }
	event := ReviewGatewayEvent{
		EventID:   "evt_atomic",
		MessageID: "om_atomic",
		EventType: "message",
		ChatID:    "oc_review",
		ChatType:  "group",
		UserID:    "ou_owner",
		Content:   "查看 owner/repo PR #42",
	}
	first := queue.Enqueue(context.Background(), event)
	if !first.Accepted || first.Job == nil {
		t.Fatalf("first atomic enqueue = %#v", first)
	}
	second := queue.Enqueue(context.Background(), event)
	if second.Accepted || !second.Duplicate || second.Reason != "duplicate_event" {
		t.Fatalf("duplicate atomic enqueue = %#v", second)
	}

	orphanEvent := event
	orphanEvent.EventID = "evt_orphan"
	orphanEvent.MessageID = "om_orphan"
	orphanKey := reviewGatewayDedupeKey(orphanEvent)
	reserved, err := store.ReserveContext(context.Background(), orphanKey, now, 24*time.Hour)
	if err != nil || !reserved {
		t.Fatalf("create P2.0 orphan reservation = %v, %v", reserved, err)
	}
	recovered := queue.Enqueue(context.Background(), orphanEvent)
	if !recovered.Accepted || recovered.Job == nil {
		t.Fatalf("orphan reservation was not recovered atomically: %#v", recovered)
	}
	var jobCount int
	if err := store.db.QueryRow(
		"SELECT COUNT(*) FROM review_gateway_jobs WHERE dedupe_key=?",
		orphanKey,
	).Scan(&jobCount); err != nil {
		t.Fatalf("count recovered job: %v", err)
	}
	if jobCount != 1 {
		t.Fatalf("recovered job count = %d", jobCount)
	}
}

func TestSQLiteReviewGatewayStorePersistsResultAndReplyState(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-result.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "job-result")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker-result",
		Now:        now,
		Limit:      1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:    reviewGatewayResultSchema,
		JobID:            job.JobID,
		Status:           "completed",
		Mode:             "preview",
		Action:           job.Action,
		Repository:       job.Repository,
		PRNumber:         job.PRNumber,
		RequestedBy:      job.RequestedBy,
		ReadOnlyGitLink:  true,
		MutatesGitLink:   false,
		CompletedAt:      now.Format(time.RFC3339),
		CollectionStatus: "complete",
		HeadSHA:          "head-sha",
	}
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	var status, resultJSON, replyStatus string
	if err := store.db.QueryRow(
		"SELECT status, result_json, reply_status FROM review_gateway_jobs WHERE job_id=?",
		job.JobID,
	).Scan(&status, &resultJSON, &replyStatus); err != nil {
		t.Fatalf("read persisted result: %v", err)
	}
	if status != "completed" || replyStatus != "pending" || !strings.Contains(resultJSON, `"head_sha":"head-sha"`) {
		t.Fatalf("persisted state = status=%q reply=%q result=%s", status, replyStatus, resultJSON)
	}
	pending, err := store.ClaimPendingReplies(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "reply-worker",
		Now:        time.Now().UTC().Add(time.Second),
		Limit:      1,
	})
	if err != nil || len(pending) != 1 {
		t.Fatalf("ClaimPendingReplies = %#v, %v", pending, err)
	}
	if pending[0].Result.HeadSHA != "head-sha" || pending[0].Job.SourceMessageID != job.SourceMessageID {
		t.Fatalf("pending reply = %#v", pending[0])
	}
}

func TestSQLiteReviewGatewayStoreRetriesThenPublishesTerminalFailure(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-retry.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "job-retry")
	job.MaxAttempts = 2
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claim := func(at time.Time, owner string) ReviewGatewayJob {
		t.Helper()
		jobs, claimErr := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
			LeaseOwner: owner,
			Now:        at,
			Limit:      1,
		})
		if claimErr != nil || len(jobs) != 1 {
			t.Fatalf("ClaimReadyJobs(%s) = %#v, %v", owner, jobs, claimErr)
		}
		return jobs[0]
	}
	first := claim(now, "worker-1")
	failed := failedReviewGatewayResult(first, errors.New("temporary read failure"), now)
	willRetry, err := store.RetryOrFailJob(context.Background(), first, failed, failed.Error, now)
	if err != nil || !willRetry {
		t.Fatalf("first RetryOrFailJob = retry=%v err=%v", willRetry, err)
	}
	tooEarly, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker-early",
		Now:        now.Add(time.Second),
		Limit:      1,
	})
	if err != nil || len(tooEarly) != 0 {
		t.Fatalf("retry was claimable before next_attempt_at: %#v, %v", tooEarly, err)
	}
	second := claim(now.Add(3*time.Second), "worker-2")
	if second.AttemptCount != 2 {
		t.Fatalf("second attempt count = %d", second.AttemptCount)
	}
	failed = failedReviewGatewayResult(second, errors.New("terminal read failure"), now.Add(3*time.Second))
	willRetry, err = store.RetryOrFailJob(context.Background(), second, failed, failed.Error, now.Add(3*time.Second))
	if err != nil || willRetry {
		t.Fatalf("terminal RetryOrFailJob = retry=%v err=%v", willRetry, err)
	}
	pending, err := store.ClaimPendingReplies(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "reply-worker",
		Now:        now.Add(4 * time.Second),
		Limit:      1,
	})
	if err != nil || len(pending) != 1 {
		t.Fatalf("terminal failure reply = %#v, %v", pending, err)
	}
	if pending[0].Result.Status != "failed" || pending[0].Result.AttemptCount != 2 {
		t.Fatalf("terminal failure result = %#v", pending[0].Result)
	}
}

func TestSQLiteReviewGatewayStoreMigratesP20Schema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "review-gateway-migrate.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	_, err = db.Exec(`
		CREATE TABLE review_gateway_events (
			dedupe_key TEXT PRIMARY KEY,
			received_at TEXT NOT NULL,
			expires_at TEXT NOT NULL
		);
		CREATE TABLE review_gateway_jobs (
			job_id TEXT PRIMARY KEY,
			dedupe_key TEXT NOT NULL,
			status TEXT NOT NULL,
			action TEXT NOT NULL,
			repository TEXT,
			pr_number INTEGER,
			requested_by TEXT NOT NULL,
			chat_id TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			error_summary TEXT
		);
		CREATE TABLE review_action_plans (
			plan_id TEXT PRIMARY KEY,
			repository TEXT NOT NULL,
			pr_number INTEGER NOT NULL,
			actor_id TEXT NOT NULL,
			gitlink_login TEXT NOT NULL,
			expected_head_sha TEXT NOT NULL,
			source_fingerprint TEXT NOT NULL,
			review_status TEXT NOT NULL,
			content TEXT NOT NULL,
			status TEXT NOT NULL,
			idempotency_key TEXT NOT NULL UNIQUE,
			source_job_id TEXT NOT NULL,
			review_id TEXT NOT NULL DEFAULT '',
			error_summary TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`)
	if err != nil {
		t.Fatalf("create P2.0 schema: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close P2.0 db: %v", err)
	}
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("migrate P2.0 schema: %v", err)
	}
	defer store.Close()
	var resultJSONColumn string
	if err := store.db.QueryRow(
		"SELECT name FROM pragma_table_info('review_gateway_jobs') WHERE name='result_json'",
	).Scan(&resultJSONColumn); err != nil {
		t.Fatalf("result_json migration missing: %v", err)
	}
	for _, column := range []string{
		"installation_id",
		"source_chat_id",
		"lease_owner",
		"lease_expires_at",
		"attempt_count",
		"max_attempts",
		"reconciliation_status",
	} {
		var migrated string
		if err := store.db.QueryRow(
			"SELECT name FROM pragma_table_info('review_action_plans') WHERE name=?",
			column,
		).Scan(&migrated); err != nil {
			t.Fatalf("review_action_plans.%s migration missing: %v", column, err)
		}
	}
}

func TestReviewGatewayHandlerBudgetStopsLockedSQLite(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "review-gateway-locked.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_review",
			Repository: "owner/repo",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{}, store)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 1, nil)
	tx, err := store.db.Begin()
	if err != nil {
		t.Fatalf("lock db: %v", err)
	}
	defer tx.Rollback()
	started := time.Now()
	err = handleReviewGatewayInbound(
		context.Background(),
		ReviewGatewayEvent{
			EventID:   "evt_locked",
			MessageID: "om_locked",
			EventType: "message",
			ChatID:    "oc_review",
			UserID:    "ou_owner",
			Content:   "查看 owner/repo PR #42",
		},
		queue,
		nil,
		&reviewGatewayJSONOutput{writer: io.Discard},
		300*time.Millisecond,
		100*time.Millisecond,
	)
	elapsed := time.Since(started)
	if err == nil || !strings.Contains(err.Error(), "persistence failed") {
		t.Fatalf("locked handler error = %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Fatalf("handler exceeded controlled budget: %v", elapsed)
	}
}

func TestReviewGatewayLiveHandlerLogsOnlyHashedIdentifiers(t *testing.T) {
	store := NewMemoryReviewGatewayJobStore()
	gateway, err := NewReviewGateway(ReviewGatewayBindings{
		Bindings: []ReviewChatBinding{{
			ChatID:     "oc_sensitive_chat",
			Repository: "owner/repo",
			Enabled:    true,
		}},
	}, ReviewGatewayConfig{}, nil)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 1, nil)
	var outputBuffer bytes.Buffer
	output := &reviewGatewayJSONOutput{writer: &outputBuffer}
	replies := NewReviewGatewayReplyDispatcher(&recordingReviewGatewaySender{}, store, output, 1)
	replies.instanceID = "instance-test"
	event := ReviewGatewayEvent{
		EventID:   "evt_sensitive_event",
		MessageID: "om_sensitive_message",
		EventType: "message",
		ChatID:    "oc_sensitive_chat",
		UserID:    "ou_sensitive_user",
		Content:   "查看 owner/repo PR #42",
	}
	if err := handleReviewGatewayInbound(
		context.Background(),
		event,
		queue,
		replies,
		output,
		2*time.Second,
		500*time.Millisecond,
	); err != nil {
		t.Fatalf("handleReviewGatewayInbound: %v", err)
	}
	logged := outputBuffer.String()
	for _, raw := range []string{event.EventID, event.MessageID, event.ChatID, event.UserID} {
		if strings.Contains(logged, raw) {
			t.Fatalf("raw Feishu identifier leaked to live output: %s", raw)
		}
	}
	if !strings.Contains(logged, reviewGatewayHashIdentifier(event.MessageID)) {
		t.Fatalf("hashed message ID missing from live output: %s", logged)
	}
}

func TestReviewGatewayLiveSenderUsesDirectAPIWithoutIdentifierLogs(t *testing.T) {
	var requestBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/v3/tenant_access_token/internal":
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","tenant_access_token":"tenant-test","expire":7200}`)
		case "/im/v1/messages/om_sensitive_source/reply":
			if got := r.Header.Get("Authorization"); got != "Bearer tenant-test" {
				t.Errorf("authorization = %q", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
				t.Errorf("decode message body: %v", err)
			}
			_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{"message_id":"om_result","chat_id":"oc_sensitive_chat"}}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	oldWriter := log.Writer()
	oldFlags := log.Flags()
	oldPrefix := log.Prefix()
	var logBuffer bytes.Buffer
	log.SetOutput(&logBuffer)
	log.SetFlags(0)
	log.SetPrefix("")
	defer func() {
		log.SetOutput(oldWriter)
		log.SetFlags(oldFlags)
		log.SetPrefix(oldPrefix)
	}()

	sender := &reviewGatewayLiveSender{
		client:    OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()},
		appID:     "cli_test",
		appSecret: "secret-test",
	}
	result, err := sender.Send(context.Background(), &larktypes.SendInput{
		ChatID:         "oc_sensitive_chat",
		ReplyMessageID: "om_sensitive_source",
		MsgType:        "text",
		Text:           "accepted",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if result.MessageID != "om_result" || result.ChatID != "oc_sensitive_chat" {
		t.Fatalf("unexpected send result: %#v", result)
	}
	if requestBody["msg_type"] != "text" || !strings.Contains(requestBody["content"], "accepted") {
		t.Fatalf("unexpected request body: %#v", requestBody)
	}
	if logged := logBuffer.String(); strings.Contains(logged, "oc_sensitive_chat") || strings.Contains(logged, "om_sensitive_source") {
		t.Fatalf("raw Feishu identifier leaked to standard logger: %s", logged)
	}
}

func TestReviewGatewayAsyncOutputDoesNotBlockHandler(t *testing.T) {
	writer := &blockingReviewGatewayWriter{
		started: make(chan struct{}, 1),
		release: make(chan struct{}),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	output := &reviewGatewayJSONOutput{writer: writer}
	output.Start(ctx, 1)
	if !output.TryEmit(map[string]string{"event": "first"}) {
		t.Fatal("first output was not accepted")
	}
	select {
	case <-writer.started:
	case <-time.After(time.Second):
		t.Fatal("writer did not start")
	}
	if !output.TryEmit(map[string]string{"event": "second"}) {
		t.Fatal("second output should fit buffered queue")
	}
	started := time.Now()
	if output.TryEmit(map[string]string{"event": "third"}) {
		t.Fatal("third output should be dropped instead of blocking")
	}
	if time.Since(started) > 50*time.Millisecond {
		t.Fatal("non-blocking output path blocked")
	}
	close(writer.release)
}

func TestReviewGatewayReplyDispatcherRepliesOnceToOriginalMessage(t *testing.T) {
	now := time.Date(2026, 7, 30, 10, 0, 0, 0, time.UTC)
	store := NewMemoryReviewGatewayJobStore()
	job := testReviewGatewayJob(now, "job-reply")
	if err := store.SaveJob(context.Background(), job); err != nil {
		t.Fatalf("SaveJob: %v", err)
	}
	claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
		LeaseOwner: "worker",
		Now:        now,
		Limit:      1,
	})
	if err != nil || len(claimed) != 1 {
		t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
	}
	result := ReviewGatewayExecutionResult{
		SchemaVersion:    reviewGatewayResultSchema,
		JobID:            job.JobID,
		Status:           "completed",
		Mode:             "preview",
		Action:           job.Action,
		Repository:       job.Repository,
		PRNumber:         job.PRNumber,
		RequestedBy:      job.RequestedBy,
		ReadOnlyGitLink:  true,
		MutatesGitLink:   false,
		CompletedAt:      now.Format(time.RFC3339),
		CollectionStatus: "complete",
		ReviewStage:      "triaged",
		Decision:         "pending",
	}
	if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
		t.Fatalf("CompleteJob: %v", err)
	}
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(
		sender,
		store,
		&reviewGatewayJSONOutput{writer: io.Discard},
		2,
	)
	dispatcher.now = func() time.Time { return now.Add(time.Second) }
	dispatcher.deliverPendingReplies(context.Background())
	dispatcher.deliverPendingReplies(context.Background())

	sender.mu.Lock()
	defer sender.mu.Unlock()
	if len(sender.inputs) != 1 {
		t.Fatalf("reply count = %d, want 1", len(sender.inputs))
	}
	if sender.inputs[0].ReplyMessageID != job.SourceMessageID {
		t.Fatalf("reply target = %q", sender.inputs[0].ReplyMessageID)
	}
	if sender.inputs[0].MsgType != "interactive" ||
		!strings.Contains(sender.inputs[0].Card, "GitLink 写入：0") {
		t.Fatalf("reply card boundary missing: %#v", sender.inputs[0])
	}
	store.mu.Lock()
	replyStatus := store.ReplyStatus[job.JobID]
	store.mu.Unlock()
	if replyStatus != "sent" {
		t.Fatalf("reply status = %q", replyStatus)
	}
}

func TestReviewGatewayReplyDispatcherMaintainsOneCardPerWorkItem(t *testing.T) {
	now := time.Date(2026, 8, 1, 11, 0, 0, 0, time.UTC)
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "fixed-card.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	sender := &recordingReviewGatewaySender{}
	dispatcher := NewReviewGatewayReplyDispatcher(
		sender,
		store,
		&reviewGatewayJSONOutput{writer: io.Discard},
		2,
	)
	// CompleteJob timestamps reply readiness with the real clock. Keep the
	// dispatcher clock ahead of that value so this test remains deterministic.
	dispatcher.now = func() time.Time { return time.Now().Add(time.Minute) }

	deliver := func(id, decision string, at time.Time) ReviewCollaborationBundle {
		t.Helper()
		job := testReviewGatewayJob(at, id)
		if err := store.SaveJob(context.Background(), job); err != nil {
			t.Fatalf("SaveJob: %v", err)
		}
		claimed, err := store.ClaimReadyJobs(context.Background(), ReviewGatewayClaimOptions{
			LeaseOwner: "worker", Now: at, Limit: 1,
		})
		if err != nil || len(claimed) != 1 {
			t.Fatalf("ClaimReadyJobs = %#v, %v", claimed, err)
		}
		result := fullReviewGatewayResultFixture()
		result.JobID = job.JobID
		result.Action = job.Action
		result.Decision = decision
		result.CompletedAt = at.UTC().Format(time.RFC3339)
		item := ReviewCollaborationItem{
			SchemaVersion:       reviewCollaborationItemSchema,
			PRKey:               reviewCollaborationScopeKey(job.InstallationID, job.ChatID, "owner/repo", 42),
			InstallationID:      job.InstallationID,
			Repository:          "owner/repo",
			PRNumber:            42,
			ChatID:              "oc_review",
			ReviewStage:         "triaged",
			Decision:            decision,
			CollectionStatus:    "complete",
			CollaborationStatus: "claimed",
			AssignedTo:          "ou_owner",
			UpdatedAt:           at.UTC().Format(time.RFC3339),
		}
		bundle := BuildReviewCollaborationBundle(item)
		bundle.Card = buildReviewGatewayResultCard(job, result, &item)
		result.Collaboration = &bundle
		result.ResultCard = bundle.Card
		if err := store.CompleteJob(context.Background(), claimed[0], result); err != nil {
			t.Fatalf("CompleteJob: %v", err)
		}
		dispatcher.deliverPendingReplies(context.Background())
		return bundle
	}

	deliver("card-first", "pending", now)
	latest := deliver("card-second", "approved", now.Add(time.Minute))
	sender.mu.Lock()
	sendCount := len(sender.inputs)
	updateCount := len(sender.updates)
	var notification larktypes.SendInput
	if len(sender.inputs) > 1 {
		notification = sender.inputs[1]
	}
	updatedMessageID := ""
	if len(sender.updatedMessageIDs) > 0 {
		updatedMessageID = sender.updatedMessageIDs[0]
	}
	sender.mu.Unlock()
	if sendCount != 2 || updateCount != 1 || updatedMessageID != "om_reply" {
		t.Fatalf("fixed card calls = send:%d update:%d message:%q", sendCount, updateCount, updatedMessageID)
	}
	if notification.MsgType != "text" || notification.ReplyMessageID != "om_card-second" ||
		notification.Text == "" || notification.Card != "" ||
		!strings.Contains(notification.Text, "正式卡片已更新") {
		t.Fatalf("fixed card current-message notification = %#v", notification)
	}
	state, err := store.GetReviewResourceState(context.Background(), latest.UniqueKey, "feishu_card")
	if err != nil {
		t.Fatalf("GetReviewResourceState: %v", err)
	}
	if state.RemoteID != "om_reply" || state.ContentFingerprint != reviewGatewayCardFingerprint(latest.Card) {
		t.Fatalf("fixed card state = %#v", state)
	}
}

func TestOpenAPIClientPatchesInteractiveMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPatch || request.URL.Path != "/im/v1/messages/om_card" {
			t.Fatalf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Authorization") != "Bearer tenant-token" {
			t.Fatalf("authorization = %q", request.Header.Get("Authorization"))
		}
		var body map[string]string
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		if !strings.Contains(body["content"], `"header"`) {
			t.Fatalf("message content = %q", body["content"])
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"code":0,"msg":"ok"}`))
	}))
	defer server.Close()
	client := NewOpenAPIClient(server.Client())
	client.BaseURL = server.URL
	if err := client.PatchInteractiveMessage(
		context.Background(),
		"tenant-token",
		"om_card",
		baseCard("Review", "blue", nil),
	); err != nil {
		t.Fatalf("PatchInteractiveMessage: %v", err)
	}
}

func TestGitLinkWebhookIngressVerifiesSignatureDeduplicatesAndQueuesRefresh(t *testing.T) {
	t.Setenv("TEST_GITLINK_WEBHOOK_SECRET", "webhook-secret")
	now := time.Date(2026, 8, 1, 12, 0, 0, 0, time.UTC)
	bindings := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "main",
			OperationMode:       "collaborate",
			AllowedRepositories: []string{"owner/repo"},
			WebhookSecretRef:    "env:TEST_GITLINK_WEBHOOK_SECRET",
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{{
			ChatID: "oc_review", InstallationID: "main",
			Repositories: []string{"owner/repo"}, Enabled: true,
		}},
	}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "webhook.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	gateway, err := NewReviewGateway(bindings, ReviewGatewayConfig{Now: func() time.Time { return now }}, store)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	queue := NewReviewGatewayQueue(gateway, store, 2, nil)
	queue.now = func() time.Time { return now }
	ingress, err := NewGitLinkWebhookIngress(bindings, queue)
	if err != nil {
		t.Fatalf("NewGitLinkWebhookIngress: %v", err)
	}
	ingress.now = func() time.Time { return now }
	payload := []byte(`{"action":"synchronize","repository":{"full_name":"owner/repo"},"pull_request":{"number":42}}`)
	mac := hmac.New(sha256.New, []byte("webhook-secret"))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))
	deliver := func(signature string) (int, GitLinkWebhookIngressResult) {
		t.Helper()
		request := httptest.NewRequest(http.MethodPost, "/gitlink/events", strings.NewReader(string(payload)))
		request.Header.Set("X-Gitea-Event", "pull_request")
		request.Header.Set("X-Gitea-Delivery", "delivery-42")
		request.Header.Set("X-Gitea-Signature", signature)
		response := httptest.NewRecorder()
		ingress.ServeHTTP(response, request)
		var result GitLinkWebhookIngressResult
		if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatalf("decode webhook response: %v", err)
		}
		return response.Code, result
	}
	status, result := deliver(signature)
	if status != http.StatusAccepted || !result.Accepted || result.Queued != 1 || result.Repository != "owner/repo" {
		t.Fatalf("first webhook result = status:%d %#v", status, result)
	}
	var payloadJSON string
	if err := store.db.QueryRow("SELECT payload_json FROM review_gateway_jobs").Scan(&payloadJSON); err != nil {
		t.Fatalf("read queued webhook job: %v", err)
	}
	var job ReviewGatewayJob
	if err := json.Unmarshal([]byte(payloadJSON), &job); err != nil {
		t.Fatalf("decode queued webhook job: %v", err)
	}
	if job.Action != "refresh_review_context" || !job.NotifyChat || job.MutatesGitLink || job.PRNumber != 42 {
		t.Fatalf("queued webhook job = %#v", job)
	}
	status, result = deliver(signature)
	if status != http.StatusAccepted || !result.Duplicate || result.Queued != 0 {
		t.Fatalf("duplicate webhook result = status:%d %#v", status, result)
	}
	status, result = deliver(strings.Repeat("0", sha256.Size*2))
	if status != http.StatusUnauthorized || result.Reason != "signature_verification_failed" {
		t.Fatalf("invalid signature result = status:%d %#v", status, result)
	}
}

func TestReviewWebhookListenAddressDefaultsToLoopback(t *testing.T) {
	if err := validateReviewWebhookListenAddress("127.0.0.1:8787", false); err != nil {
		t.Fatalf("loopback address rejected: %v", err)
	}
	if err := validateReviewWebhookListenAddress(":8787", false); err == nil {
		t.Fatal("public wildcard address accepted without explicit override")
	}
	if err := validateReviewWebhookListenAddress(":8787", true); err != nil {
		t.Fatalf("explicit public override rejected: %v", err)
	}
}

func TestReviewGatewayChannelPolicyRequiresPreboundGroup(t *testing.T) {
	bindings := ReviewGatewayBindings{Bindings: []ReviewChatBinding{
		{ChatID: "oc_enabled", Repository: "owner/repo", Enabled: true},
		{ChatID: "oc_disabled", Repository: "owner/other", Enabled: false},
	}}
	channel := newFeishuReviewGatewayChannel("app-id", "app-secret", bindings, nil, "test-instance")
	policy := channel.GetPolicy()
	if len(policy.GroupAllowlist) != 1 || policy.GroupAllowlist[0] != "oc_enabled" {
		t.Fatalf("group allowlist = %#v", policy.GroupAllowlist)
	}
	if policy.RequireMention == nil || !*policy.RequireMention || policy.DMMode != "disabled" {
		t.Fatalf("channel policy = %#v", policy)
	}
}

func TestReviewGatewayBotIdentityFetchDoesNotLogRawOpenID(t *testing.T) {
	client := lark.NewClient(
		"app-id",
		"app-secret",
		lark.WithLogLevel(larkcore.LogLevelWarn),
		lark.WithHttpClient(reviewGatewayMockHTTPClient{do: func(request *http.Request) (*http.Response, error) {
			body := `{"code":0,"msg":"success"}`
			switch request.URL.Path {
			case "/open-apis/auth/v3/tenant_access_token/internal":
				body = `{"code":0,"msg":"success","tenant_access_token":"tenant-test-token","expire":7200}`
			case "/open-apis/bot/v3/info":
				body = `{"code":0,"msg":"success","bot":{"open_id":"ou_sensitive_bot_id","app_name":"gitlink","activate_status":2}}`
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}}),
	)
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	originalStdout := os.Stdout
	os.Stdout = writePipe
	defer func() { os.Stdout = originalStdout }()
	identity, err := fetchReviewGatewayBotIdentity(context.Background(), client)
	_ = writePipe.Close()
	os.Stdout = originalStdout
	logged, _ := io.ReadAll(readPipe)
	_ = readPipe.Close()
	if err != nil || identity == nil || identity.OpenID != "ou_sensitive_bot_id" {
		t.Fatalf("identity=%#v err=%v", identity, err)
	}
	if strings.Contains(string(logged), "ou_sensitive_bot_id") {
		t.Fatalf("bot open_id leaked to stdout: %s", logged)
	}
}

func TestReviewGatewayInstanceLockRejectsSecondActiveProcess(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "gateway.db")
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	first, err := acquireReviewGatewayInstanceLock(statePath, "cli_test", now)
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	defer first.Release()
	if _, err := acquireReviewGatewayInstanceLock(statePath, "cli_test", now.Add(time.Second)); err == nil ||
		!strings.Contains(err.Error(), "another review gateway instance is active") {
		t.Fatalf("second lock error = %v", err)
	}
}

func TestReviewGatewayInstanceLockCanBeReleasedAndReacquired(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "gateway.db")
	first, err := acquireReviewGatewayInstanceLock(statePath, "cli_test", time.Now())
	if err != nil {
		t.Fatalf("first lock: %v", err)
	}
	if err := first.Release(); err != nil {
		t.Fatalf("release lock: %v", err)
	}
	second, err := acquireReviewGatewayInstanceLock(statePath, "cli_test", time.Now())
	if err != nil {
		t.Fatalf("second lock after release: %v", err)
	}
	defer second.Release()
	if first.metadata.InstanceID == second.metadata.InstanceID {
		t.Fatal("instance IDs should differ")
	}
}

func TestReviewGatewayObservationHashesIdentityAndOmitsContent(t *testing.T) {
	message := &larktypes.NormalizedMessage{
		EventID:      "evt_secret",
		MessageID:    "om_secret",
		ChatID:       "oc_secret",
		ChatType:     "group",
		UserID:       "ou_secret",
		Content:      "查看 PR #431 and a private note",
		MentionedBot: true,
	}
	observation := reviewGatewayNormalizedObservation("policy", "instance-test", message)
	observation.Allowed = reviewGatewayBoolPointer(true)
	payload, err := json.Marshal(observation)
	if err != nil {
		t.Fatalf("marshal observation: %v", err)
	}
	text := string(payload)
	for _, secret := range []string{"evt_secret", "om_secret", "oc_secret", "ou_secret", "private note"} {
		if strings.Contains(text, secret) {
			t.Fatalf("observation leaked %q: %s", secret, text)
		}
	}
	if observation.EventIDHash == "" || observation.MessageIDHash == "" || observation.ChatIDHash == "" {
		t.Fatalf("missing hashes: %#v", observation)
	}
}

func TestReviewGatewayRejectionNoticeIsSafeAndBounded(t *testing.T) {
	for _, reason := range []string{"sender_not_allowed", "binding_requires_admin", "repository_qualification_required", "unsupported_read_only_command"} {
		text := formatReviewGatewayNotice(reason)
		if text == "" {
			t.Fatalf("empty notice for %s", reason)
		}
		if strings.Contains(strings.ToLower(text), "token") || strings.Contains(text, "oc_") || strings.Contains(text, "ou_") {
			t.Fatalf("unsafe notice for %s: %s", reason, text)
		}
	}
	if text := formatReviewGatewayNotice("chat_not_bound"); text != "" {
		t.Fatalf("unbound chat should remain silent: %s", text)
	}
}

func TestReviewCollaborationClaimDeadlineReleaseAndAudit(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "collaboration.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "collaboration")
	job.Action = "claim_review"
	job.Mode = "collaboration"
	presentationResult := fullReviewGatewayResultFixture()
	presentationResult.CompletedAt = now.Format(time.RFC3339)
	if _, err := store.UpsertCollaborationFacts(context.Background(), job, presentationResult, now); err != nil {
		t.Fatalf("seed complete presentation: %v", err)
	}
	item, err := store.ApplyCollaborationAction(context.Background(), job, now)
	if err != nil {
		t.Fatalf("claim: %v", err)
	}
	if item.AssignedTo != job.RequestedBy || item.CollaborationStatus != "reviewing" {
		t.Fatalf("claimed item = %#v", item)
	}

	job.Action = "set_review_deadline"
	job.Argument = "2026-08-02"
	job.JobID = "job-deadline"
	item, err = store.ApplyCollaborationAction(context.Background(), job, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("deadline: %v", err)
	}
	if item.DueAt != "2026-08-02" {
		t.Fatalf("deadline = %q", item.DueAt)
	}

	job.Action = "release_review"
	job.JobID = "job-release"
	item, err = store.ApplyCollaborationAction(context.Background(), job, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("release: %v", err)
	}
	if item.AssignedTo != "" || item.CollaborationStatus != "unassigned" || item.DueAt != "" {
		t.Fatalf("released item = %#v", item)
	}
	var auditCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_collaboration_audit WHERE pr_key = ?`, item.PRKey).Scan(&auditCount); err != nil {
		t.Fatalf("audit count: %v", err)
	}
	if auditCount != 3 {
		t.Fatalf("audit count = %d, want 3", auditCount)
	}
}

func TestReviewCollaborationScopesStateByInstallationAndChat(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "collaboration-scope.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC)
	result := fullReviewGatewayResultFixture()
	result.CompletedAt = now.Format(time.RFC3339)
	result.HeadSHA = "head-installation-a"
	result.SourceFingerprint = "fingerprint-a"
	jobA := testReviewGatewayJob(now, "scope-a")
	jobA.InstallationID = "installation-a"
	jobA.ChatID = "chat-a"
	jobA.RequestedBy = "ou_reviewer_a"
	if _, err := store.UpsertCollaborationFacts(context.Background(), jobA, result, now); err != nil {
		t.Fatalf("upsert chat A facts: %v", err)
	}
	jobA.Action = "claim_review"
	itemA, err := store.ApplyCollaborationAction(context.Background(), jobA, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim chat A: %v", err)
	}

	jobB := jobA
	jobB.JobID = "job-scope-b"
	jobB.ChatID = "chat-b"
	jobB.RequestedBy = "ou_reviewer_b"
	jobB.Action = "read_review_context"
	itemB, err := store.UpsertCollaborationFacts(context.Background(), jobB, result, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("upsert chat B facts: %v", err)
	}
	if itemB.AssignedTo != "" || itemB.CollaborationStatus != "unassigned" {
		t.Fatalf("chat B inherited chat A collaboration: %#v", itemB)
	}
	jobB.Action = "claim_review"
	itemB, err = store.ApplyCollaborationAction(context.Background(), jobB, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("claim chat B: %v", err)
	}
	if itemA.PRKey == itemB.PRKey || BuildReviewCollaborationBundle(itemA).UniqueKey == BuildReviewCollaborationBundle(itemB).UniqueKey {
		t.Fatalf("chat-scoped identities collided: A=%#v B=%#v", itemA, itemB)
	}

	jobC := jobA
	jobC.JobID = "job-scope-installation-b"
	jobC.InstallationID = "installation-b"
	jobC.Action = "read_review_context"
	result.HeadSHA = "head-installation-b"
	result.SourceFingerprint = "fingerprint-b"
	result.CompletedAt = now.Add(4 * time.Minute).Format(time.RFC3339)
	itemC, err := store.UpsertCollaborationFacts(context.Background(), jobC, result, now.Add(4*time.Minute))
	if err != nil {
		t.Fatalf("upsert installation B facts: %v", err)
	}
	if itemC.HeadSHA != "head-installation-b" || itemC.PRKey == itemA.PRKey || itemC.AssignedTo != "" {
		t.Fatalf("installation B inherited installation A state: %#v", itemC)
	}

	itemsA, err := store.ListCollaborationItems(context.Background(), "installation-a", "chat-a", jobA.Repository, "")
	if err != nil || len(itemsA) != 1 || itemsA[0].AssignedTo != "ou_reviewer_a" {
		t.Fatalf("chat A list = %#v, %v", itemsA, err)
	}
	itemsB, err := store.ListCollaborationItems(context.Background(), "installation-a", "chat-b", jobB.Repository, "")
	if err != nil || len(itemsB) != 1 || itemsB[0].AssignedTo != "ou_reviewer_b" {
		t.Fatalf("chat B list = %#v, %v", itemsB, err)
	}
}

func TestReviewCollaborationLegacyMigrationIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy-collaboration.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create legacy store: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO chat_repository_bindings (
		chat_id, installation_id, repository, enabled, updated_at
	) VALUES ('chat-legacy', 'installation-legacy', 'owner/repo', 1, '2026-08-04T00:00:00Z')`)
	if err != nil {
		t.Fatalf("insert legacy binding: %v", err)
	}
	_, err = store.db.Exec(`INSERT INTO review_collaboration_items (
		pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
	) VALUES (
		'owner/repo#42', 'owner/repo', 42, 'chat-legacy', 'human_reviewing', 'pending',
		'complete', 'head-legacy', 'fingerprint-legacy', 'ou_legacy',
		'reviewing', '2026-08-05', 0, 'ou_legacy', '2026-08-04T00:00:00Z'
	)`)
	if err != nil {
		t.Fatalf("insert legacy item: %v", err)
	}
	legacyResourceKey := stableKey("review-work-item", "owner/repo", "42")
	_, err = store.db.Exec(`INSERT INTO review_collaboration_resources (
		work_item_key, resource_type, remote_id, content_fingerprint, updated_at
	) VALUES (?, 'feishu_card', 'om_existing_card', 'fingerprint-card', '2026-08-04T00:00:00Z')`, legacyResourceKey)
	if err != nil {
		t.Fatalf("insert legacy resource mapping: %v", err)
	}
	_ = store.Close()

	for pass := 0; pass < 2; pass++ {
		store, err = OpenSQLiteReviewGatewayStore(path)
		if err != nil {
			t.Fatalf("migration pass %d: %v", pass+1, err)
		}
		var snapshots, states int
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_pr_snapshots`).Scan(&snapshots); err != nil {
			t.Fatalf("count snapshots: %v", err)
		}
		if err := store.db.QueryRow(`SELECT COUNT(*) FROM review_collaboration_states`).Scan(&states); err != nil {
			t.Fatalf("count states: %v", err)
		}
		if snapshots != 1 || states != 1 {
			t.Fatalf("migration pass %d counts = snapshots:%d states:%d", pass+1, snapshots, states)
		}
		scopedKey := stableKey(
			"review-work-item",
			reviewCollaborationScopeKey("installation-legacy", "chat-legacy", "owner/repo", 42),
		)
		resource, err := store.GetReviewResourceState(context.Background(), scopedKey, "feishu_card")
		if err != nil || resource.RemoteID != "" {
			t.Fatalf("migration pass %d copied unverified resource = %#v, %v", pass+1, resource, err)
		}
		plans, err := store.ListReviewResourceMigrations(context.Background(), "", ReviewResourceCard, "installation-legacy")
		if err != nil || len(plans) != 1 || plans[0].Status != ReviewResourceMigrationNeedsReconciliation {
			t.Fatalf("migration pass %d plans = %#v, %v", pass+1, plans, err)
		}
		_ = store.Close()
	}
}

func TestReviewCollaborationAmbiguousLegacyStateDoesNotLeakAcrossInstallations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous-legacy-collaboration.db")
	store, err := OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("create legacy store: %v", err)
	}
	for _, installationID := range []string{"installation-a", "installation-b"} {
		if _, err := store.db.Exec(`INSERT INTO chat_repository_bindings (
			chat_id, installation_id, repository, enabled, updated_at
		) VALUES ('chat-shared', ?, 'owner/repo', 1, '2026-08-04T00:00:00Z')`, installationID); err != nil {
			t.Fatalf("insert %s binding: %v", installationID, err)
		}
	}
	if _, err := store.db.Exec(`INSERT INTO review_collaboration_items (
		pr_key, repository, pr_number, chat_id, review_stage, decision,
		collection_status, head_sha, source_fingerprint, assigned_to,
		collaboration_status, due_at, archived, updated_by, updated_at
	) VALUES (
		'owner/repo#42', 'owner/repo', 42, 'chat-shared', 'human_reviewing', 'pending',
		'complete', 'head-legacy', 'fingerprint-legacy', 'ou_legacy',
		'reviewing', '2026-08-05', 0, 'ou_legacy', '2026-08-04T00:00:00Z'
	)`); err != nil {
		t.Fatalf("insert legacy item: %v", err)
	}
	_ = store.Close()

	store, err = OpenSQLiteReviewGatewayStore(path)
	if err != nil {
		t.Fatalf("open ambiguous legacy store: %v", err)
	}
	defer store.Close()
	job := testReviewGatewayJob(time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC), "ambiguous")
	job.InstallationID = "installation-a"
	job.ChatID = "chat-shared"
	result := fullReviewGatewayResultFixture()
	result.CompletedAt = time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC).Format(time.RFC3339)
	result.HeadSHA = "head-current"
	result.SourceFingerprint = "fingerprint-current"
	item, err := store.UpsertCollaborationFacts(context.Background(), job, result, time.Date(2026, 8, 4, 8, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("upsert scoped item: %v", err)
	}
	if item.AssignedTo != "" || item.CollaborationStatus != "unassigned" || item.DueAt != "" {
		t.Fatalf("ambiguous legacy state leaked into installation A: %#v", item)
	}
}

func TestReviewCollaborationPartialSnapshotPreservesCompleteFacts(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "collaboration.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "facts")
	complete := fullReviewGatewayResultFixture()
	complete.CompletedAt = now.Format(time.RFC3339)
	complete.HeadSHA = "head-complete"
	complete.SourceFingerprint = "fingerprint-complete"
	item, err := store.UpsertCollaborationFacts(context.Background(), job, complete, now)
	if err != nil {
		t.Fatalf("complete upsert: %v", err)
	}
	partial := complete
	partial.CollectionStatus = "partial"
	partial.Partial = true
	partial.HeadSHA = "head-partial"
	partial.SourceFingerprint = "fingerprint-partial"
	item, err = store.UpsertCollaborationFacts(context.Background(), job, partial, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("partial upsert: %v", err)
	}
	if item.HeadSHA != "head-complete" || item.SourceFingerprint != "fingerprint-complete" || item.CollectionStatus != "complete" {
		t.Fatalf("partial snapshot overwrote complete facts: %#v", item)
	}
}

func TestReviewCollaborationBundleUsesOneStableWorkItem(t *testing.T) {
	item := ReviewCollaborationItem{
		SchemaVersion:       reviewCollaborationItemSchema,
		PRKey:               "Gitlink/gitlink-cli#431",
		Repository:          "Gitlink/gitlink-cli",
		PRNumber:            431,
		ReviewStage:         "human_reviewing",
		Decision:            "pending",
		CollectionStatus:    "complete",
		AssignedTo:          "ou_reviewer",
		CollaborationStatus: "reviewing",
		DueAt:               "2026-08-05",
		UpdatedAt:           "2026-07-31T08:00:00Z",
	}
	bundle := BuildReviewCollaborationBundle(item)
	if bundle.GitLinkWrites != 0 || bundle.UniqueKey == "" || bundle.BitableRecord.UniqueKey != bundle.UniqueKey {
		t.Fatalf("bundle identity/boundary = %#v", bundle)
	}
	if bundle.Task == nil || bundle.Task.UniqueKey != bundle.UniqueKey {
		t.Fatalf("task not linked to work item: %#v", bundle.Task)
	}
	if bundle.Task.AssigneeOpenID != "ou_reviewer" || bundle.Task.DueDate != "2026-08-05" {
		t.Fatalf("task collaboration mapping = %#v", bundle.Task)
	}
	if !strings.Contains(bundle.DocMarkdown, "GitLink 写入：0") {
		t.Fatalf("document boundary missing: %s", bundle.DocMarkdown)
	}
}

func TestReviewResourceStatePreventsDuplicateTaskAndDocSnapshot(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "resources.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	if err := store.SaveReviewResourceState(
		context.Background(),
		"review-work-item:owner_repo_42",
		"feishu_task",
		"task-remote-1",
		"fingerprint-1",
		now,
	); err != nil {
		t.Fatalf("SaveReviewResourceState: %v", err)
	}
	state, err := store.GetReviewResourceState(
		context.Background(),
		"review-work-item:owner_repo_42",
		"feishu_task",
	)
	if err != nil {
		t.Fatalf("GetReviewResourceState: %v", err)
	}
	if state.RemoteID != "task-remote-1" || state.ContentFingerprint != "fingerprint-1" {
		t.Fatalf("resource state = %#v", state)
	}
}

func TestFeishuReviewCollaborationPublisherCreatesTaskOnce(t *testing.T) {
	taskCreates := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodPost && r.URL.Path == "/task/v2/tasks":
			taskCreates++
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"task":{"guid":"task-guid-431"}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "publisher.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	publisher := &FeishuReviewCollaborationPublisher{
		Client: OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()},
		Store:  store,
		Config: ReviewCollaborationPublisherConfig{
			AppID:      "cli_test",
			AppSecret:  "test-secret",
			EnableTask: true,
		},
		Now: func() time.Time {
			return time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
		},
	}
	bundle := ReviewCollaborationBundle{
		UniqueKey: "review-work-item:Gitlink_gitlink-cli_431",
		Task: &TaskCandidate{
			UniqueKey:   "review-work-item:Gitlink_gitlink-cli_431",
			Title:       "Review Gitlink/gitlink-cli #431",
			Description: "Human review task",
		},
	}

	first, err := publisher.Publish(context.Background(), bundle)
	if err != nil {
		t.Fatalf("first Publish: %v", err)
	}
	second, err := publisher.Publish(context.Background(), bundle)
	if err != nil {
		t.Fatalf("second Publish: %v", err)
	}
	if taskCreates != 1 {
		t.Fatalf("task creates = %d, want 1", taskCreates)
	}
	if len(first) != 1 || first[0].Action != "created" || first[0].RemoteID != "task-guid-431" {
		t.Fatalf("first publish = %#v", first)
	}
	if len(second) != 1 || second[0].Action != "unchanged" || second[0].RemoteID != "task-guid-431" {
		t.Fatalf("second publish = %#v", second)
	}
}

func TestFeishuReviewPublisherPreservesBaseHumanFieldsUnlessAuthoritative(t *testing.T) {
	searchCalls := 0
	updatedFields := []map[string]interface{}{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = writer.Write([]byte(`{"code":0,"tenant_access_token":"tenant-token","expire":7200}`))
		case request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/records/search"):
			searchCalls++
			_, _ = writer.Write([]byte(`{"code":0,"data":{"items":[{"record_id":"rec-review-42"}]}}`))
		case request.Method == http.MethodPut && strings.HasSuffix(request.URL.Path, "/records/rec-review-42"):
			var body struct {
				Fields map[string]interface{} `json:"fields"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode Base update: %v", err)
			}
			updatedFields = append(updatedFields, body.Fields)
			_, _ = writer.Write([]byte(`{"code":0,"data":{"record":{"record_id":"rec-review-42"}}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "base-lifecycle.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	publisher := &FeishuReviewCollaborationPublisher{
		Client: OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()},
		Store:  store,
		Config: ReviewCollaborationPublisherConfig{
			AppID: "cli_test", AppSecret: "secret", BaseAppToken: "base", ReviewTableID: "table",
		},
		Now: func() time.Time { return time.Date(2026, 8, 1, 13, 0, 0, 0, time.UTC) },
	}
	item := ReviewCollaborationItem{
		SchemaVersion: reviewCollaborationItemSchema, PRKey: "owner/repo#42",
		Repository: "owner/repo", PRNumber: 42, ReviewStage: "triaged",
		Decision: "pending", CollectionStatus: "complete", AssignedTo: "ou_existing",
		CollaborationStatus: "claimed", DueAt: "2026-08-05",
		UpdatedAt: "2026-08-01T13:00:00Z",
	}
	firstBundle := BuildReviewCollaborationBundle(item)
	first, err := publisher.Publish(context.Background(), firstBundle)
	if err != nil || len(first) != 1 || first[0].Action != "updated" {
		t.Fatalf("first Base publish = %#v, err=%v", first, err)
	}
	for _, manual := range []string{"assigned_to", "collaboration_status", "due_at"} {
		if _, exists := updatedFields[0][manual]; exists {
			t.Fatalf("GitLink fact refresh overwrote Base human field %q: %#v", manual, updatedFields[0])
		}
	}
	item.AssignedTo = "ou_new"
	item.DueAt = "2026-08-06"
	item.UpdatedAt = "2026-08-01T13:01:00Z"
	secondBundle := BuildReviewCollaborationBundle(item)
	secondBundle.HumanFieldsAuthoritative = true
	second, err := publisher.Publish(context.Background(), secondBundle)
	if err != nil || len(second) != 1 || second[0].Action != "updated" {
		t.Fatalf("authoritative Base publish = %#v, err=%v", second, err)
	}
	if updatedFields[1]["assigned_to"] != "ou_new" || updatedFields[1]["due_at"] != "2026-08-06" {
		t.Fatalf("authoritative human fields missing: %#v", updatedFields[1])
	}
	if searchCalls != 1 {
		t.Fatalf("Base unique key search calls = %d, want one initial reconciliation", searchCalls)
	}
}

func TestFeishuReviewPublisherUpdatesAndCompletesExistingTask(t *testing.T) {
	createCount := 0
	completedValues := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = writer.Write([]byte(`{"code":0,"tenant_access_token":"tenant-token","expire":7200}`))
		case request.Method == http.MethodPost && request.URL.Path == "/task/v2/tasks":
			createCount++
			_, _ = writer.Write([]byte(`{"code":0,"data":{"task":{"guid":"task-guid-42"}}}`))
		case request.Method == http.MethodPatch && request.URL.Path == "/task/v2/tasks/task-guid-42":
			var body struct {
				Task map[string]interface{} `json:"task"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatalf("decode Task patch: %v", err)
			}
			completedValues = append(completedValues, fmt.Sprint(body.Task["completed_at"]))
			_, _ = writer.Write([]byte(`{"code":0,"data":{"task":{"guid":"task-guid-42"}}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "task-lifecycle.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 1, 13, 30, 0, 0, time.UTC)
	publisher := &FeishuReviewCollaborationPublisher{
		Client: OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()},
		Store:  store, Config: ReviewCollaborationPublisherConfig{
			AppID: "cli_test", AppSecret: "secret", EnableTask: true,
		},
		Now: func() time.Time { return now },
	}
	item := ReviewCollaborationItem{
		SchemaVersion: reviewCollaborationItemSchema, PRKey: "owner/repo#42",
		Repository: "owner/repo", PRNumber: 42, ReviewStage: "triaged",
		Decision: "pending", CollectionStatus: "complete",
		CollaborationStatus: "unassigned", UpdatedAt: now.Format(time.RFC3339),
	}
	created, err := publisher.Publish(context.Background(), BuildReviewCollaborationBundle(item))
	if err != nil || created[0].Action != "created" {
		t.Fatalf("create task = %#v, err=%v", created, err)
	}
	item.Decision = "changes_pending"
	item.DueAt = "2026-08-05"
	item.UpdatedAt = now.Add(time.Minute).Format(time.RFC3339)
	updated, err := publisher.Publish(context.Background(), BuildReviewCollaborationBundle(item))
	if err != nil || updated[0].Action != "updated" {
		t.Fatalf("update task = %#v, err=%v", updated, err)
	}
	item.Archived = true
	item.UpdatedAt = now.Add(2 * time.Minute).Format(time.RFC3339)
	completed, err := publisher.Publish(context.Background(), BuildReviewCollaborationBundle(item))
	if err != nil || completed[0].Action != "completed" {
		t.Fatalf("complete task = %#v, err=%v", completed, err)
	}
	if createCount != 1 || len(completedValues) != 2 || completedValues[0] != "0" || completedValues[1] == "0" {
		t.Fatalf("Task lifecycle calls = creates:%d completed_at:%#v", createCount, completedValues)
	}
}

func TestFeishuReviewPublisherMaintainsOneDocumentPerPR(t *testing.T) {
	documentCreates := 0
	blockAppends := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodPost && request.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = writer.Write([]byte(`{"code":0,"tenant_access_token":"tenant-token","expire":7200}`))
		case request.Method == http.MethodPost && request.URL.Path == "/docx/v1/documents":
			documentCreates++
			_, _ = writer.Write([]byte(`{"code":0,"data":{"document":{"document_id":"doc-pr-42","title":"Review"}}}`))
		case request.Method == http.MethodPost && request.URL.Path == "/docx/v1/documents/doc-pr-42/blocks/doc-pr-42/children":
			blockAppends++
			_, _ = writer.Write([]byte(`{"code":0,"data":{"revision_id":2}}`))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "doc-lifecycle.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	publisher := &FeishuReviewCollaborationPublisher{
		Client: OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()},
		Store:  store, Config: ReviewCollaborationPublisherConfig{
			AppID: "cli_test", AppSecret: "secret", DocumentFolderToken: "folder-review",
		},
		Now: time.Now,
	}
	item := ReviewCollaborationItem{
		SchemaVersion: reviewCollaborationItemSchema, PRKey: "owner/repo#42",
		Repository: "owner/repo", PRNumber: 42, ReviewStage: "triaged",
		Decision: "pending", CollectionStatus: "complete",
		CollaborationStatus: "unassigned", UpdatedAt: "2026-08-01T14:00:00Z",
	}
	bundle := BuildReviewCollaborationBundle(item)
	first, err := publisher.Publish(context.Background(), bundle)
	if err != nil || first[0].Action != "created_and_appended" || first[0].RemoteID != "doc-pr-42" {
		t.Fatalf("first document publish = %#v, err=%v", first, err)
	}
	second, err := publisher.Publish(context.Background(), bundle)
	if err != nil || second[0].Action != "unchanged" {
		t.Fatalf("unchanged document publish = %#v, err=%v", second, err)
	}
	item.Decision = "approved"
	item.UpdatedAt = "2026-08-01T14:01:00Z"
	third, err := publisher.Publish(context.Background(), BuildReviewCollaborationBundle(item))
	if err != nil || third[0].Action != "appended" || third[0].RemoteID != "doc-pr-42" {
		t.Fatalf("updated document publish = %#v, err=%v", third, err)
	}
	if documentCreates != 1 || blockAppends != 2 {
		t.Fatalf("document lifecycle calls = creates:%d appends:%d", documentCreates, blockAppends)
	}
}

func TestReviewExecutorCollectsPublisherWarningsWithoutRetryingGitLinkRead(t *testing.T) {
	result := ReviewGatewayExecutionResult{Status: "completed"}
	publisher := &recordingReviewCollaborationPublisher{
		results: []ReviewResourceSyncResult{{
			Resource: "feishu_doc",
			Action:   "failed",
			Error:    "permission denied",
		}},
	}
	executor := &ReviewGatewayExecutor{Publisher: publisher}
	executor.publishCollaboration(
		context.Background(),
		&result,
		ReviewCollaborationBundle{UniqueKey: "review-work-item:owner_repo_42"},
	)
	if publisher.calls != 1 || len(result.ResourceSync) != 1 || len(result.Warnings) != 1 {
		t.Fatalf("publisher outcome = %#v, calls=%d", result, publisher.calls)
	}
	if result.Status != "completed" {
		t.Fatalf("collaboration resource failure should not retry GitLink read: %#v", result)
	}
}

func TestReviewActionPlanIsIdempotentCommonOnlyAndActorBound(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "action-plan")
	plan := NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Review summary",
		now,
	)
	first, err := store.CreateReviewActionPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	second, err := store.CreateReviewActionPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("idempotent CreateReviewActionPlan: %v", err)
	}
	if first.PlanID != second.PlanID || first.ReviewStatus != "common" ||
		first.MutationStatus != reviewMutationNone {
		t.Fatalf("idempotent plans = %#v / %#v", first, second)
	}
	differentInstallation := job
	differentInstallation.InstallationID = "other"
	installationPlan := NewReviewActionPlan(
		differentInstallation,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Review summary",
		now,
	)
	differentChat := job
	differentChat.ChatID = "oc_other"
	chatPlan := NewReviewActionPlan(
		differentChat,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Review summary",
		now,
	)
	if installationPlan.IdempotencyKey == plan.IdempotencyKey || chatPlan.IdempotencyKey == plan.IdempotencyKey {
		t.Fatal("ActionPlan idempotency must include installation and source chat scope")
	}
	approved := plan
	approved.PlanID = "approved-plan"
	approved.IdempotencyKey = "approved-key"
	approved.ReviewStatus = "approved"
	if _, err := store.CreateReviewActionPlan(context.Background(), approved); err == nil {
		t.Fatal("approved action plan must be rejected")
	}
	if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       "another-user",
		LeaseOwner:    "test-lease",
		Now:           now,
		LeaseDuration: time.Minute,
	}); err == nil {
		t.Fatal("another Feishu user must not claim the action plan")
	}
}

func TestReviewActionPlanLeaseRecoversOnlyBeforeRemoteWriteBoundary(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan-lease.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "action-plan-lease")
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Review summary",
		now,
	))
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	first, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       job.RequestedBy,
		LeaseOwner:    "lease-one",
		Now:           now,
		LeaseDuration: time.Minute,
	})
	if err != nil || first.AttemptCount != 1 || first.Reconciliation != "pre_write" {
		t.Fatalf("first claim = %#v, err=%v", first, err)
	}
	if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       job.RequestedBy,
		LeaseOwner:    "lease-two",
		Now:           now.Add(30 * time.Second),
		LeaseDuration: time.Minute,
	}); err == nil {
		t.Fatal("active execution lease must not be stolen")
	}
	recovered, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       job.RequestedBy,
		LeaseOwner:    "lease-two",
		Now:           now.Add(2 * time.Minute),
		LeaseDuration: time.Minute,
	})
	if err != nil || recovered.AttemptCount != 2 || recovered.LeaseOwner != "lease-two" {
		t.Fatalf("recovered claim = %#v, err=%v", recovered, err)
	}
	if err := store.MarkReviewActionPlanWriteStarted(
		context.Background(),
		plan.PlanID,
		"lease-two",
		now.Add(2*time.Minute),
	); err != nil {
		t.Fatalf("MarkReviewActionPlanWriteStarted: %v", err)
	}
	postBoundary, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if err != nil || postBoundary.MutationStatus != reviewMutationPossible {
		t.Fatalf("post-boundary mutation state = %#v, err=%v", postBoundary, err)
	}
	if _, err := store.ClaimReviewActionPlan(context.Background(), ReviewActionPlanClaimOptions{
		PlanID:        plan.PlanID,
		ActorID:       job.RequestedBy,
		LeaseOwner:    "lease-three",
		Now:           now.Add(4 * time.Minute),
		LeaseDuration: time.Minute,
	}); err == nil || !strings.Contains(err.Error(), "reconciliation") {
		t.Fatalf("post-boundary reclaim error = %v", err)
	}
}

func TestFormatReviewGatewayResultReplyDistinguishesUncertainWrite(t *testing.T) {
	reply := formatReviewGatewayResultReply(ReviewGatewayJob{}, ReviewGatewayExecutionResult{
		WriteResult: &ReviewWriteResult{
			Status:         "unknown_needs_reconciliation",
			Repository:     "owner/repo",
			PRNumber:       42,
			MutationStatus: reviewMutationPossible,
			Reconciliation: "query current reviews before retrying",
		},
	})
	if !strings.Contains(reply, "写入：结果不确定") ||
		!strings.Contains(reply, "禁止自动重试") || strings.Contains(reply, "写入：0") {
		t.Fatalf("uncertain write reply = %q", reply)
	}
}

func TestReviewWriteRemainsDisabledWithoutExplicitStartupFlag(t *testing.T) {
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "action-plan-disabled")
	plan := NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Review summary",
		now,
	)
	plan, err = store.CreateReviewActionPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-disabled")
	confirmJob.Action = "confirm_common_review"
	confirmJob.Argument = plan.PlanID
	confirmJob.Mode = "controlled_write"
	executor := &ReviewGatewayExecutor{
		Runtime:       &common.RuntimeContext{},
		ActionPlans:   store,
		Installations: testReviewGatewayInstallations(),
		IdentityBindings: []ReviewIdentityBinding{{
			FeishuUserID: confirmJob.RequestedBy,
			GitLinkLogin: "gitlink-reviewer",
			Enabled:      true,
		}},
		EnableGitLinkWrite: false,
	}
	result, err := executor.Execute(context.Background(), confirmJob)
	if err != nil {
		t.Fatalf("disabled confirmation: %v", err)
	}
	if result.WriteResult == nil || result.WriteResult.Status != "write_disabled" ||
		result.WriteResult.Mutated || result.WriteResult.MutationStatus != reviewMutationNone || result.MutatesGitLink {
		t.Fatalf("write boundary = %#v", result)
	}
	stored, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if err != nil {
		t.Fatalf("GetReviewActionPlan: %v", err)
	}
	if stored.Status != "pending_confirmation" {
		t.Fatalf("disabled confirmation changed plan state to %q", stored.Status)
	}
}

func TestConfirmedCommonReviewRejectsScopeMismatchBeforePOST(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()

	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan-scope.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	sourceJob := testReviewGatewayJob(now, "scope-plan")
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
		sourceJob,
		"gitlink-reviewer",
		"head-431",
		"fingerprint-431",
		"Evidence-backed Review summary",
		now,
	))
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	runtime := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}

	tests := []struct {
		name      string
		configure func(*ReviewGatewayJob, map[string]GitLinkInstallation)
	}{
		{
			name: "cross installation with the same login",
			configure: func(job *ReviewGatewayJob, installations map[string]GitLinkInstallation) {
				job.InstallationID = "other"
				installations["other"] = GitLinkInstallation{
					InstallationID:      "other",
					OperationMode:       "write",
					AllowedRepositories: []string{"owner/repo"},
					Enabled:             true,
				}
			},
		},
		{
			name: "cross chat",
			configure: func(job *ReviewGatewayJob, _ map[string]GitLinkInstallation) {
				job.ChatID = "oc_other"
			},
		},
		{
			name: "repository removed from source chat",
			configure: func(job *ReviewGatewayJob, _ map[string]GitLinkInstallation) {
				job.Repositories = []string{"owner/other"}
			},
		},
		{
			name: "repository removed from installation allowlist",
			configure: func(_ *ReviewGatewayJob, installations map[string]GitLinkInstallation) {
				installation := installations["test"]
				installation.AllowedRepositories = []string{"owner/other"}
				installations["test"] = installation
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-"+strings.ReplaceAll(test.name, " ", "-"))
			confirmJob.Action = "confirm_common_review"
			confirmJob.Argument = plan.PlanID
			confirmJob.Mode = "controlled_write"
			installations := testReviewGatewayInstallations()
			test.configure(&confirmJob, installations)
			executor := &ReviewGatewayExecutor{
				Runtime:       runtime,
				ActionPlans:   store,
				Installations: installations,
				IdentityBindings: []ReviewIdentityBinding{
					{InstallationID: "test", FeishuUserID: sourceJob.RequestedBy, GitLinkLogin: "gitlink-reviewer", Enabled: true},
					{InstallationID: "other", FeishuUserID: sourceJob.RequestedBy, GitLinkLogin: "gitlink-reviewer", Enabled: true},
				},
				EnableGitLinkWrite: true,
				Now:                func() time.Time { return now.Add(time.Minute) },
			}
			if _, err := executor.Execute(context.Background(), confirmJob); err == nil {
				t.Fatal("scope mismatch must be rejected")
			}
			if writes := state.writes(); writes != 0 {
				t.Fatalf("scope mismatch created %d Reviews", writes)
			}
			stored, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
			if err != nil {
				t.Fatalf("GetReviewActionPlan: %v", err)
			}
			if stored.Status != "pending_confirmation" || stored.MutationStatus != reviewMutationNone {
				t.Fatalf("scope rejection changed ActionPlan: %#v", stored)
			}
		})
	}
}

func TestCardConfirmationCannotCrossSourceChat(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan-card-scope.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 8, 2, 8, 0, 0, 0, time.UTC)
	sourceJob := testReviewGatewayJob(now, "card-scope-plan")
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
		sourceJob, "gitlink-reviewer", "head-431", "fingerprint-431", "Review summary", now,
	))
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	bindings := ReviewGatewayBindings{
		SchemaVersion: reviewGatewayBindingSchema,
		Installations: []GitLinkInstallation{{
			InstallationID:      "test",
			GitLinkHost:         server.URL,
			CredentialRef:       "env:TEST_GITLINK_TOKEN",
			OperationMode:       "write",
			AllowedRepositories: []string{"owner/repo"},
			Enabled:             true,
		}},
		Bindings: []ReviewChatBinding{
			{ChatID: "oc_review", InstallationID: "test", Repositories: []string{"owner/repo"}, Enabled: true},
			{ChatID: "oc_other", InstallationID: "test", Repositories: []string{"owner/repo"}, Enabled: true},
		},
	}
	gateway, err := NewReviewGateway(bindings, ReviewGatewayConfig{}, store)
	if err != nil {
		t.Fatalf("NewReviewGateway: %v", err)
	}
	event, ok := reviewGatewayEventFromCardAction(&larktypes.CardActionEvent{
		EventID:   "evt_cross_chat_confirm",
		MessageID: "om_cross_chat_confirm",
		ChatID:    "oc_other",
		Operator:  larktypes.CardActionOperator{OpenID: sourceJob.RequestedBy},
		Action: larktypes.CardActionPayload{Value: map[string]interface{}{
			"command": fmt.Sprintf("确认 Review %s", plan.PlanID),
		}},
	})
	if !ok {
		t.Fatal("card confirmation was not normalized")
	}
	receipt, err := gateway.Plan(event)
	if err != nil || !receipt.Accepted || receipt.Job == nil {
		t.Fatalf("cross-chat card plan = %#v, err=%v", receipt, err)
	}
	executor := &ReviewGatewayExecutor{
		Runtime:       &common.RuntimeContext{Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL}},
		ActionPlans:   store,
		Installations: testReviewGatewayInstallations(),
		IdentityBindings: []ReviewIdentityBinding{{
			InstallationID: "test",
			FeishuUserID:   sourceJob.RequestedBy,
			GitLinkLogin:   "gitlink-reviewer",
			Enabled:        true,
		}},
		EnableGitLinkWrite: true,
		Now:                func() time.Time { return now.Add(time.Minute) },
	}
	if _, err := executor.Execute(context.Background(), *receipt.Job); err == nil {
		t.Fatal("cross-chat card confirmation must be rejected")
	}
	if writes := state.writes(); writes != 0 {
		t.Fatalf("cross-chat card confirmation created %d Reviews", writes)
	}
}

func TestConfirmedCommonReviewWritesExactlyOnceAndVerifiesReadBack(t *testing.T) {
	var writeCount, readbackCount int
	var writeMu sync.Mutex
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/owner/repo/pulls/42.json":
			_, _ = writer.Write([]byte(`{"pull_request":{"number":42,"title":"Review me","state":"open","head_commit_sha":"head-431"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/versions.json":
			_, _ = writer.Write([]byte(`{"versions":[{"id":7,"head_commit_sha":"head-431","files_count":1,"commits_count":1}]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/owner/repo/pulls/42/files.json":
			_, _ = writer.Write([]byte(`{"files":[{"filename":"main.go","additions":3}]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/reviews.json":
			writeMu.Lock()
			written := writeCount > 0
			if written {
				readbackCount++
			}
			writeMu.Unlock()
			if written {
				_, _ = writer.Write([]byte(`{"reviews":[{"id":901,"status":"common","commit_id":"head-431","content":"Evidence-backed Review summary","user":{"login":"gitlink-reviewer"}}]}`))
			} else {
				_, _ = writer.Write([]byte(`{"reviews":[]}`))
			}
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/journals.json":
			_, _ = writer.Write([]byte(`{"journals":[]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/users/me.json":
			_, _ = writer.Write([]byte(`{"login":"gitlink-reviewer","id":71}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v1/owner/repo/pulls/42/reviews.json":
			var payload map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode Review payload: %v", err)
			}
			if payload["status"] != "common" || payload["commit_id"] != "head-431" {
				t.Fatalf("Review payload = %#v", payload)
			}
			writeMu.Lock()
			writeCount++
			writeMu.Unlock()
			_, _ = writer.Write([]byte(`{"review":{"id":901}}`))
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
	defer server.Close()

	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "action-plan.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "live-write")
	runtime := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}
	initialContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner:           "owner",
		Repo:            "repo",
		Number:          42,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext for plan: %v", err)
	}
	plan := NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		"head-431",
		initialContext.WorkItem.SourceFingerprint,
		"Evidence-backed Review summary",
		now,
	)
	plan, err = store.CreateReviewActionPlan(context.Background(), plan)
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	executor := &ReviewGatewayExecutor{
		Runtime:       runtime,
		ActionPlans:   store,
		Installations: testReviewGatewayInstallations(),
		IdentityBindings: []ReviewIdentityBinding{{
			FeishuUserID: job.RequestedBy,
			GitLinkLogin: "gitlink-reviewer",
			Enabled:      true,
		}},
		EnableGitLinkWrite: true,
		Now:                func() time.Time { return now.Add(time.Minute) },
	}
	confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-live")
	confirmJob.Action = "confirm_common_review"
	confirmJob.Argument = plan.PlanID
	confirmJob.Mode = "controlled_write"
	result, err := executor.Execute(context.Background(), confirmJob)
	if err != nil {
		t.Fatalf("confirm common Review: %v", err)
	}
	writeMu.Lock()
	writes, readbacks := writeCount, readbackCount
	writeMu.Unlock()
	if result.WriteResult == nil || !result.WriteResult.Mutated ||
		result.WriteResult.MutationStatus != reviewMutationConfirmed ||
		result.WriteResult.Status != "completed" || result.WriteResult.ReviewID != "901" ||
		result.WriteResult.Reconciliation != "verified by GitLink GET read-back" ||
		!result.MutatesGitLink || writes != 1 || readbacks != 1 {
		t.Fatalf("write result = %#v, writes=%d, readbacks=%d", result, writes, readbacks)
	}
	if _, err := executor.Execute(context.Background(), confirmJob); err == nil {
		t.Fatal("completed ActionPlan must reject a second confirmation")
	}
	writeMu.Lock()
	writes = writeCount
	writeMu.Unlock()
	if writes != 1 {
		t.Fatalf("duplicate confirmation created %d Reviews", writes)
	}
}

func TestConfirmedCommonReviewRequiresStrictReadBack(t *testing.T) {
	for _, mode := range []string{"missing_id", "missing", "wrong_status", "wrong_head", "wrong_content", "wrong_actor", "error"} {
		t.Run(mode, func(t *testing.T) {
			state := &commonReviewTestServerState{readbackMode: mode}
			server := newCommonReviewTestServer(t, state)
			defer server.Close()
			runtime := &common.RuntimeContext{
				Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
				Owner:  "owner",
				Repo:   "repo",
			}
			initialContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
				Owner:           "owner",
				Repo:            "repo",
				Number:          42,
				VersionLimit:    100,
				ThreadLimit:     100,
				IncludePR:       true,
				IncludeFiles:    true,
				IncludeVersions: true,
				IncludeReviews:  true,
				IncludeThreads:  true,
			})
			if err != nil {
				t.Fatalf("FetchReviewContext for plan: %v", err)
			}
			store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "readback-"+mode+".db"))
			if err != nil {
				t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
			}
			defer store.Close()
			now := time.Date(2026, 8, 2, 9, 0, 0, 0, time.UTC)
			job := testReviewGatewayJob(now, "readback-plan-"+mode)
			plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
				job,
				"gitlink-reviewer",
				"head-431",
				initialContext.WorkItem.SourceFingerprint,
				"Evidence-backed Review summary",
				now,
			))
			if err != nil {
				t.Fatalf("CreateReviewActionPlan: %v", err)
			}
			executor := &ReviewGatewayExecutor{
				Runtime:       runtime,
				ActionPlans:   store,
				Installations: testReviewGatewayInstallations(),
				IdentityBindings: []ReviewIdentityBinding{{
					InstallationID: "test",
					FeishuUserID:   job.RequestedBy,
					GitLinkLogin:   "gitlink-reviewer",
					Enabled:        true,
				}},
				EnableGitLinkWrite: true,
				Now:                func() time.Time { return now.Add(time.Minute) },
			}
			confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-readback-"+mode)
			confirmJob.Action = "confirm_common_review"
			confirmJob.Argument = plan.PlanID
			confirmJob.Mode = "controlled_write"
			result, err := executor.Execute(context.Background(), confirmJob)
			if err != nil {
				t.Fatalf("confirm common Review: %v", err)
			}
			expectedReviewID := "902"
			if mode == "missing_id" {
				expectedReviewID = ""
			}
			if result.WriteResult == nil || result.WriteResult.Status != "unknown_needs_reconciliation" ||
				result.WriteResult.MutationStatus != reviewMutationConfirmed || !result.WriteResult.Mutated ||
				result.WriteResult.ReviewID != expectedReviewID || !result.MutatesGitLink {
				t.Fatalf("read-back mismatch result = %#v", result)
			}
			stored, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
			if err != nil {
				t.Fatalf("GetReviewActionPlan: %v", err)
			}
			if stored.Status != "unknown" || stored.Reconciliation != "required" ||
				stored.MutationStatus != reviewMutationConfirmed || stored.ReviewID != expectedReviewID {
				t.Fatalf("read-back mismatch ActionPlan = %#v", stored)
			}
			if _, err := executor.Execute(context.Background(), confirmJob); err == nil {
				t.Fatal("unreconciled confirmed write must not be retried")
			}
			if writes := state.writes(); writes != 1 {
				t.Fatalf("read-back mismatch created %d Reviews", writes)
			}
		})
	}
}

func TestConfirmedCommonReviewRejectsChangedSourceFingerprint(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()
	runtime := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}
	initialContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner:           "owner",
		Repo:            "repo",
		Number:          42,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext for plan: %v", err)
	}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "fingerprint.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "fingerprint-plan")
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		initialContext.CurrentHeadSHA,
		initialContext.WorkItem.SourceFingerprint,
		"Review based on initial facts",
		now,
	))
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	state.mu.Lock()
	state.reviewChanged = true
	state.mu.Unlock()
	executor := &ReviewGatewayExecutor{
		Runtime:       runtime,
		ActionPlans:   store,
		Installations: testReviewGatewayInstallations(),
		IdentityBindings: []ReviewIdentityBinding{{
			FeishuUserID: job.RequestedBy,
			GitLinkLogin: "gitlink-reviewer",
			Enabled:      true,
		}},
		EnableGitLinkWrite: true,
		Now:                func() time.Time { return now.Add(time.Minute) },
	}
	confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-fingerprint")
	confirmJob.Action = "confirm_common_review"
	confirmJob.Argument = plan.PlanID
	confirmJob.Mode = "controlled_write"
	result, err := executor.Execute(context.Background(), confirmJob)
	if err != nil {
		t.Fatalf("confirm changed fingerprint: %v", err)
	}
	if result.WriteResult == nil || result.WriteResult.Status != "stale" ||
		result.WriteResult.Mutated || result.WriteResult.MutationStatus != reviewMutationNone || state.writes() != 0 {
		t.Fatalf("fingerprint boundary = %#v, writes=%d", result, state.writes())
	}
}

func TestReviewWriteCompletionPersistenceFailureRequiresReconciliation(t *testing.T) {
	state := &commonReviewTestServerState{}
	server := newCommonReviewTestServer(t, state)
	defer server.Close()
	runtime := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
	}
	initialContext, err := workflow.FetchReviewContext(runtime, workflow.ReviewContextOptions{
		Owner:           "owner",
		Repo:            "repo",
		Number:          42,
		VersionLimit:    100,
		ThreadLimit:     100,
		IncludePR:       true,
		IncludeFiles:    true,
		IncludeVersions: true,
		IncludeReviews:  true,
		IncludeThreads:  true,
	})
	if err != nil {
		t.Fatalf("FetchReviewContext for plan: %v", err)
	}
	store, err := OpenSQLiteReviewGatewayStore(filepath.Join(t.TempDir(), "finish-failure.db"))
	if err != nil {
		t.Fatalf("OpenSQLiteReviewGatewayStore: %v", err)
	}
	defer store.Close()
	now := time.Date(2026, 7, 31, 8, 0, 0, 0, time.UTC)
	job := testReviewGatewayJob(now, "finish-failure-plan")
	plan, err := store.CreateReviewActionPlan(context.Background(), NewReviewActionPlan(
		job,
		"gitlink-reviewer",
		initialContext.CurrentHeadSHA,
		initialContext.WorkItem.SourceFingerprint,
		"Review requiring durable completion",
		now,
	))
	if err != nil {
		t.Fatalf("CreateReviewActionPlan: %v", err)
	}
	actionPlans := &failingFinishReviewActionPlanStore{ReviewActionPlanStore: store}
	executor := &ReviewGatewayExecutor{
		Runtime:       runtime,
		ActionPlans:   actionPlans,
		Installations: testReviewGatewayInstallations(),
		IdentityBindings: []ReviewIdentityBinding{{
			FeishuUserID: job.RequestedBy,
			GitLinkLogin: "gitlink-reviewer",
			Enabled:      true,
		}},
		EnableGitLinkWrite: true,
		Now:                func() time.Time { return now.Add(time.Minute) },
	}
	confirmJob := testReviewGatewayJob(now.Add(time.Minute), "confirm-finish-failure")
	confirmJob.Action = "confirm_common_review"
	confirmJob.Argument = plan.PlanID
	confirmJob.Mode = "controlled_write"
	result, err := executor.Execute(context.Background(), confirmJob)
	if err != nil {
		t.Fatalf("completion persistence failure must not retry POST: %v", err)
	}
	if result.WriteResult == nil ||
		result.WriteResult.Status != "unknown_needs_reconciliation" ||
		!result.WriteResult.Mutated ||
		result.WriteResult.MutationStatus != reviewMutationConfirmed ||
		state.writes() != 1 ||
		actionPlans.unknownCalls != 1 {
		t.Fatalf("reconciliation result = %#v, writes=%d, unknown=%d", result, state.writes(), actionPlans.unknownCalls)
	}
	stored, err := store.GetReviewActionPlan(context.Background(), plan.PlanID)
	if err != nil {
		t.Fatalf("GetReviewActionPlan: %v", err)
	}
	if stored.Status != "unknown" || stored.Reconciliation != "required" {
		t.Fatalf("stored reconciliation state = %#v", stored)
	}
}

func testReviewGatewayJob(now time.Time, id string) ReviewGatewayJob {
	return ReviewGatewayJob{
		SchemaVersion:    reviewGatewayJobSchema,
		JobID:            id,
		DedupeKey:        "feishu:message:" + id,
		Status:           "queued",
		Mode:             "preview",
		Action:           "read_review_context",
		InstallationID:   "test",
		InstallationMode: "write",
		Repository:       "owner/repo",
		Repositories:     []string{"owner/repo"},
		PRNumber:         42,
		ChatID:           "oc_review",
		RequestedBy:      "ou_owner",
		SourceEventID:    "evt_" + id,
		SourceMessageID:  "om_" + id,
		CreatedAt:        now.Format(time.RFC3339Nano),
		MutatesGitLink:   false,
		MaxAttempts:      3,
		NextAttemptAt:    now.Format(time.RFC3339Nano),
	}
}

func testReviewGatewayInstallations() map[string]GitLinkInstallation {
	return map[string]GitLinkInstallation{
		"test": {
			InstallationID:      "test",
			OperationMode:       "write",
			AllowedRepositories: []string{"owner/repo"},
			Enabled:             true,
		},
	}
}

type commonReviewTestServerState struct {
	mu             sync.Mutex
	reviewChanged  bool
	readbackMode   string
	writeCount     int
	postedContent  string
	postedCommitID string
	postedStatus   string
}

func (s *commonReviewTestServerState) writes() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeCount
}

func newCommonReviewTestServer(t *testing.T, state *commonReviewTestServerState) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/owner/repo/pulls/42.json":
			_, _ = writer.Write([]byte(`{"pull_request":{"number":42,"title":"Review me","state":"open","head_commit_sha":"head-431"}}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/versions.json":
			_, _ = writer.Write([]byte(`{"versions":[{"id":7,"head_commit_sha":"head-431","files_count":1,"commits_count":1}]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/owner/repo/pulls/42/files.json":
			_, _ = writer.Write([]byte(`{"files":[{"filename":"main.go","additions":3}]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/reviews.json":
			state.mu.Lock()
			changed := state.reviewChanged
			mode := state.readbackMode
			written := state.writeCount > 0
			content := state.postedContent
			commitID := state.postedCommitID
			status := state.postedStatus
			state.mu.Unlock()
			if changed {
				_, _ = writer.Write([]byte(`{"reviews":[{"id":88,"status":"common","commit_id":"head-431","content":"new fact","created_at":"2026-07-31T08:00:30Z"}]}`))
				break
			}
			if !written || mode == "missing" {
				_, _ = writer.Write([]byte(`{"reviews":[]}`))
				break
			}
			if mode == "error" {
				http.Error(writer, `{"message":"read-back unavailable"}`, http.StatusServiceUnavailable)
				break
			}
			actor := "gitlink-reviewer"
			switch mode {
			case "wrong_status":
				status = "approved"
			case "wrong_head":
				commitID = "another-head"
			case "wrong_content":
				content = "different content"
			case "wrong_actor":
				actor = "another-reviewer"
			}
			_ = json.NewEncoder(writer).Encode(map[string]interface{}{
				"reviews": []map[string]interface{}{{
					"id":        902,
					"status":    status,
					"commit_id": commitID,
					"content":   content,
					"user":      map[string]interface{}{"login": actor},
				}},
			})
		case request.Method == http.MethodGet && request.URL.Path == "/v1/owner/repo/pulls/42/journals.json":
			_, _ = writer.Write([]byte(`{"journals":[]}`))
		case request.Method == http.MethodGet && request.URL.Path == "/users/me.json":
			_, _ = writer.Write([]byte(`{"login":"gitlink-reviewer","id":71}`))
		case request.Method == http.MethodPost && request.URL.Path == "/v1/owner/repo/pulls/42/reviews.json":
			var payload map[string]interface{}
			if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
				t.Fatalf("decode Review payload: %v", err)
			}
			state.mu.Lock()
			state.writeCount++
			mode := state.readbackMode
			state.postedContent, _ = payload["content"].(string)
			state.postedCommitID, _ = payload["commit_id"].(string)
			state.postedStatus, _ = payload["status"].(string)
			state.mu.Unlock()
			if mode == "missing_id" {
				_, _ = writer.Write([]byte(`{"review":{}}`))
			} else {
				_, _ = writer.Write([]byte(`{"review":{"id":902}}`))
			}
		default:
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
	}))
}

type failingFinishReviewActionPlanStore struct {
	ReviewActionPlanStore
	unknownCalls int
}

func (s *failingFinishReviewActionPlanStore) FinishReviewActionPlan(
	context.Context,
	string,
	string,
	string,
	string,
	string,
	time.Time,
) error {
	return errors.New("simulated local completion persistence failure")
}

func (s *failingFinishReviewActionPlanStore) MarkReviewActionPlanUnknown(
	ctx context.Context,
	planID,
	errorSummary string,
	mutationStatus string,
	now time.Time,
) error {
	s.unknownCalls++
	return s.ReviewActionPlanStore.MarkReviewActionPlanUnknown(ctx, planID, errorSummary, mutationStatus, now)
}

type recordingReviewGatewaySender struct {
	mu                sync.Mutex
	inputs            []larktypes.SendInput
	updates           []Card
	updatedMessageIDs []string
}

func (s *recordingReviewGatewaySender) UpdateInteractiveMessage(_ context.Context, messageID string, card Card) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updatedMessageIDs = append(s.updatedMessageIDs, messageID)
	s.updates = append(s.updates, card)
	return nil
}

func (s *recordingReviewGatewaySender) Send(_ context.Context, input *larktypes.SendInput) (*larktypes.SendResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inputs = append(s.inputs, *input)
	return &larktypes.SendResult{
		MessageID: "om_reply",
		ChatID:    input.ChatID,
	}, nil
}

type blockingReviewGatewayWriter struct {
	started chan struct{}
	release chan struct{}
}

type recordingReviewCollaborationPublisher struct {
	calls   int
	results []ReviewResourceSyncResult
	err     error
}

func (p *recordingReviewCollaborationPublisher) Publish(
	_ context.Context,
	_ ReviewCollaborationBundle,
) ([]ReviewResourceSyncResult, error) {
	p.calls++
	return p.results, p.err
}

func (w *blockingReviewGatewayWriter) Write(payload []byte) (int, error) {
	select {
	case w.started <- struct{}{}:
	default:
	}
	<-w.release
	return len(payload), nil
}

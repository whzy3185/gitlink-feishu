package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func TestShortcutsExposeExpectedCommands(t *testing.T) {
	got := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		got[shortcut.Name] = true
	}
	for _, name := range []string{"bot-test", "notify", "weekly-report", "owner-digest", "contributor-digest", "doc-export", "bitable-schema", "bitable-records", "bitable-sync", "task-preview", "task-create"} {
		if !got[name] {
			t.Fatalf("Shortcuts missing %s", name)
		}
	}
}

func TestDeliveryOptionsRejectSendDryRun(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{
		"send":        "true",
		"dry-run":     "true",
		"webhook-url": "https://open.feishu.cn/open-apis/bot/v2/hook/test",
	}}
	_, err := deliveryOptionsFromContext(ctx)
	if err == nil {
		t.Fatal("expected --send --dry-run error")
	}
}

func TestDeliveryOptionsRequireWebhookForSend(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{"send": "true"}}
	_, err := deliveryOptionsFromContext(ctx)
	if err == nil {
		t.Fatal("expected missing webhook error")
	}
}

func TestRedactWebhookURL(t *testing.T) {
	got := redactWebhookURL("https://open.feishu.cn/open-apis/bot/v2/hook/12345678-1234-1234-1234-123456789abc")
	if strings.Contains(got, "1234-1234") || !strings.Contains(got, "https://open.feishu.cn/.../") {
		t.Fatalf("redacted webhook URL leaked too much: %s", got)
	}
}

func TestRedactTokenAndResourceURL(t *testing.T) {
	if got := redactToken("abcdef1234567890"); got != "abcd...7890" {
		t.Fatalf("redactToken = %q", got)
	}
	got := redactResourceURL("https://tenant.feishu.cn/wiki/NodeToken123456789?from=copy")
	if strings.Contains(got, "Token123") || strings.Contains(got, "from=copy") {
		t.Fatalf("redactResourceURL leaked token or query: %s", got)
	}
}

func TestSignCustomBotRequestIsDeterministic(t *testing.T) {
	first := SignCustomBotRequest(1710000000, "secret")
	second := SignCustomBotRequest(1710000000, "secret")
	if first == "" || first != second {
		t.Fatalf("signature not deterministic: first=%q second=%q", first, second)
	}
}

func TestBuildWorkflowCardIncludesDocButton(t *testing.T) {
	report, err := readWorkflowReport(filepath.Join("..", "workflow", "testdata", "repo_report.json"), "en")
	if err != nil {
		t.Fatalf("readWorkflowReport returned error: %v", err)
	}
	card := BuildWorkflowCard(report, parseList(defaultInclude), "", "en", "https://example.feishu.cn/wiki/node")
	encoded, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if !strings.Contains(string(encoded), "Open Feishu report") {
		t.Fatalf("card missing doc button: %s", string(encoded))
	}
}

func TestWebhookClientSendsPayload(t *testing.T) {
	var sawTimestamp bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		var payload WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		sawTimestamp = payload.Timestamp != "" && payload.Sign != ""
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":0,"msg":"success"}`))
	}))
	defer server.Close()

	client := WebhookClient{
		URL:    server.URL,
		Secret: "secret",
		HTTP:   server.Client(),
		Now:    func() time.Time { return time.Unix(1710000000, 0) },
	}
	resp, err := client.Send(context.Background(), NewInteractivePayload(BuildBotTestCard("", "", "en")))
	if err != nil {
		t.Fatalf("Send returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.Code != 0 {
		t.Fatalf("response = %+v", resp)
	}
	if !sawTimestamp {
		t.Fatal("signed payload missing timestamp/sign")
	}
}

func TestReadWorkflowReportSupportsInputFixture(t *testing.T) {
	report, err := readWorkflowReport(filepath.Join("..", "workflow", "testdata", "repo_report.json"), "en")
	if err != nil {
		t.Fatalf("readWorkflowReport returned error: %v", err)
	}
	if report.Repository != "Gitlink/gitlink-cli" || report.ReportScore == 0 {
		t.Fatalf("report = %+v", report)
	}
}

func TestReadWorkflowReportSupportsPowerShellUTF16Redirect(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "workflow", "testdata", "repo_report.json"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	utf16Data := []byte{0xFF, 0xFE}
	for _, r := range string(raw) {
		if r > 0xFFFF {
			t.Fatalf("fixture contains non-BMP rune %q", r)
		}
		utf16Data = append(utf16Data, byte(r), byte(r>>8))
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, utf16Data, 0600); err != nil {
		t.Fatalf("write UTF-16 fixture: %v", err)
	}
	report, err := readWorkflowReport(path, "en")
	if err != nil {
		t.Fatalf("readWorkflowReport returned error: %v", err)
	}
	if report.Repository != "Gitlink/gitlink-cli" {
		t.Fatalf("Repository = %q", report.Repository)
	}
}

func TestBitableSchemaAndRecords(t *testing.T) {
	report, err := readWorkflowReport(filepath.Join("..", "workflow", "testdata", "repo_report.json"), "en")
	if err != nil {
		t.Fatalf("readWorkflowReport returned error: %v", err)
	}
	schema := BuildBitableSchema(parseList("issues,prs,reports"))
	if len(schema.Tables) != 3 {
		t.Fatalf("schema table count = %d", len(schema.Tables))
	}
	records := BuildBitableRecords(report, parseList("issues,prs,reports,tasks"), "https://example.feishu.cn/wiki/node")
	if !records.DryRun {
		t.Fatal("records must be dry-run")
	}
	if len(records.Tables["reports"]) != 1 {
		t.Fatalf("reports records = %d, want 1", len(records.Tables["reports"]))
	}
	if len(records.Tables["tasks"]) == 0 {
		t.Fatal("tasks records should be generated")
	}
}

func TestOwnerAndContributorDigestMapping(t *testing.T) {
	report := workflowReportFixture(t)
	owner := BuildOwnerDigest(report, "https://tenant.feishu.cn/wiki/node")
	if owner.Role != "owner" || owner.Repository != report.Repository {
		t.Fatalf("owner digest = %+v", owner)
	}
	if owner.IssueTotal != report.IssueSummary.Total || owner.PRTotal != report.PRSummary.Total {
		t.Fatalf("owner digest counts = %+v", owner)
	}
	contributor := BuildContributorDigest(report, "")
	if contributor.Role != "contributor" {
		t.Fatalf("contributor digest role = %q", contributor.Role)
	}
	if !strings.Contains(contributor.BoundaryDescription, "not personalized") {
		t.Fatalf("contributor boundary missing personalization warning: %s", contributor.BoundaryDescription)
	}
	card := BuildOwnerDigestCard(owner, "", "en")
	encoded, err := json.Marshal(card)
	if err != nil {
		t.Fatalf("json.Marshal returned error: %v", err)
	}
	if !strings.Contains(string(encoded), "Open GitLink repository") {
		t.Fatalf("owner card missing repository button: %s", string(encoded))
	}
}

func TestTaskCandidatesAreStable(t *testing.T) {
	report := workflowReportFixture(t)
	tasks := BuildTaskCandidates(report, "https://tenant.feishu.cn/wiki/node")
	if len(tasks) == 0 {
		t.Fatal("expected task candidates")
	}
	seen := map[string]bool{}
	for _, task := range tasks {
		if task.UniqueKey == "" || seen[task.UniqueKey] {
			t.Fatalf("unstable or duplicate task key: %+v", task)
		}
		seen[task.UniqueKey] = true
		if task.Repository != report.Repository {
			t.Fatalf("task repository = %q, want %q", task.Repository, report.Repository)
		}
	}
}

func TestBitableSyncOptionsRejectSendDryRun(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{
		"send":           "true",
		"dry-run":        "true",
		"app-id":         "cli_xxx",
		"app-secret":     "secret",
		"base-app-token": "base",
	}}
	if _, err := bitableSyncOptionsFromContext(ctx); err == nil {
		t.Fatal("expected --send --dry-run error")
	}
}

func TestBitableSyncMockHTTP(t *testing.T) {
	report := workflowReportFixture(t)
	records := BuildBitableRecords(report, []string{"reports"}, "")
	var sawCreate bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodPost && r.URL.Path == "/bitable/v1/apps/base_token/tables/tbl_report/records/search":
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"items":[]}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/bitable/v1/apps/base_token/tables/tbl_report/records":
			sawCreate = true
			var payload struct {
				Fields map[string]interface{} `json:"fields"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode bitable payload: %v", err)
			}
			if payload.Fields["unique_key"] == "" {
				t.Fatalf("payload missing unique_key: %+v", payload.Fields)
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"record":{"record_id":"rec_1234567890"}}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	oldBaseURL := openAPIBaseURL
	openAPIBaseURL = server.URL
	defer func() { openAPIBaseURL = oldBaseURL }()

	opts := BitableSyncOptions{
		AppID:        "cli_xxx",
		AppSecret:    "secret",
		BaseAppToken: "base_token",
		TableIDs:     map[string]string{"reports": "tbl_report"},
		Tables:       []string{"reports"},
		Send:         true,
	}
	if err := syncBitableOrPreview(&common.RuntimeContext{}, opts, records); err != nil {
		t.Fatalf("syncBitableOrPreview returned error: %v", err)
	}
	if !sawCreate {
		t.Fatal("expected create request")
	}
}

func TestTaskCreateOptionsRejectSendDryRun(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{
		"send":       "true",
		"dry-run":    "true",
		"app-id":     "cli_xxx",
		"app-secret": "secret",
	}}
	if _, err := taskCreateOptionsFromContext(ctx); err == nil {
		t.Fatal("expected --send --dry-run error")
	}
}

func TestTaskCreateMockHTTP(t *testing.T) {
	var sawTask bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodPost && r.URL.Path == "/task/v2/tasks":
			sawTask = true
			var payload struct {
				Summary string `json:"summary"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode task payload: %v", err)
			}
			if payload.Summary == "" {
				t.Fatal("task summary is empty")
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"task":{"guid":"task_guid_123456"}}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	oldBaseURL := openAPIBaseURL
	openAPIBaseURL = server.URL
	defer func() { openAPIBaseURL = oldBaseURL }()

	tasks := []TaskCandidate{{
		UniqueKey:   "task:test",
		Title:       "Review GitLink workflow report",
		Description: "Workflow report task",
		SourceType:  "report",
		SourceKey:   "report-review",
		Repository:  "Gitlink/gitlink-cli",
		Priority:    "low",
		TaskType:    "report_review",
		Status:      "todo",
	}}
	opts := TaskCreateOptions{AppID: "cli_xxx", AppSecret: "secret", Send: true}
	if err := createTasksOrPreview(&common.RuntimeContext{}, opts, tasks); err != nil {
		t.Fatalf("createTasksOrPreview returned error: %v", err)
	}
	if !sawTask {
		t.Fatal("expected task create request")
	}
}

func TestWikiNodeTokenFromURL(t *testing.T) {
	got := wikiNodeTokenFromURL("https://tenant.feishu.cn/wiki/NodeToken123?from=from_copylink")
	if got != "NodeToken123" {
		t.Fatalf("wikiNodeTokenFromURL = %q", got)
	}
}

func TestDocExportOptionsRequireAppCredentialsForSend(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{
		"send":       "true",
		"wiki-url":   "https://tenant.feishu.cn/wiki/NodeToken123",
		"app-id":     "",
		"app-secret": "",
	}}
	_, err := docExportOptionsFromContext(ctx)
	if err == nil {
		t.Fatal("expected missing app credential error")
	}
}

func TestOpenAPIClientDocExportFlow(t *testing.T) {
	var sawBlocks bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodGet && r.URL.Path == "/wiki/v2/spaces/get_node":
			if r.URL.Query().Get("token") != "NodeToken123" {
				t.Fatalf("wiki token = %q", r.URL.Query().Get("token"))
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"node":{"space_id":"space","node_token":"NodeToken123","obj_token":"doc_token","obj_type":"docx","node_type":"origin","title":"Report"}}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/docx/v1/documents/doc_token/blocks/doc_token/children":
			sawBlocks = true
			var payload struct {
				Children []DocBlock `json:"children"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode blocks: %v", err)
			}
			if len(payload.Children) == 0 {
				t.Fatal("no blocks in payload")
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"success","data":{"revision_id":9}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()}
	token, err := client.TenantAccessToken(context.Background(), "cli_xxx", "secret")
	if err != nil {
		t.Fatalf("TenantAccessToken returned error: %v", err)
	}
	node, err := client.GetWikiNode(context.Background(), token.Value, "NodeToken123")
	if err != nil {
		t.Fatalf("GetWikiNode returned error: %v", err)
	}
	if node.ObjToken != "doc_token" || node.ObjType != "docx" {
		t.Fatalf("node = %+v", node)
	}
	blocks := BuildDocBlocks(workflowReportFixture(t), "en")
	created, err := client.CreateBlocks(context.Background(), token.Value, node.ObjToken, node.ObjToken, blocks)
	if err != nil {
		t.Fatalf("CreateBlocks returned error: %v", err)
	}
	if created.RevisionID != 9 || !sawBlocks {
		t.Fatalf("created = %+v sawBlocks=%t", created, sawBlocks)
	}
}

func workflowReportFixture(t *testing.T) workflow.RepoReportResult {
	t.Helper()
	report, err := readWorkflowReport(filepath.Join("..", "workflow", "testdata", "repo_report.json"), "en")
	if err != nil {
		t.Fatalf("readWorkflowReport returned error: %v", err)
	}
	return report
}

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
	for _, name := range []string{"bot-test", "notify", "weekly-report", "doc-export", "bitable-schema", "bitable-records"} {
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
	records := BuildBitableRecords(report, parseList("issues,prs,reports"))
	if !records.DryRun {
		t.Fatal("records must be dry-run")
	}
	if len(records.Tables["reports"]) != 1 {
		t.Fatalf("reports records = %d, want 1", len(records.Tables["reports"]))
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

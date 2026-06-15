package feishu

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestShortcutsExposeExpectedCommands(t *testing.T) {
	got := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		got[shortcut.Name] = true
	}
	for _, name := range []string{"bot-test", "notify", "weekly-report", "bitable-schema", "bitable-records"} {
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

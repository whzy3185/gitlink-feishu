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
)

func TestBootstrapReviewBasePreviewMakesNoRequests(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("preview made request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	output, err := bootstrapReviewBase(context.Background(), ReviewBaseBootstrapOptions{
		BaseName:  defaultReviewBaseName,
		TableName: defaultReviewTableName,
		ViewName:  defaultReviewViewName,
		TimeZone:  "Asia/Shanghai",
		OutputEnv: filepath.Join(t.TempDir(), "resources.ps1"),
	}, OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()})
	if err != nil {
		t.Fatalf("bootstrapReviewBase preview: %v", err)
	}
	if requests != 0 || output.FeishuWrites != 0 || output.GitLinkWrites != 0 {
		t.Fatalf("preview performed writes: requests=%d output=%+v", requests, output)
	}
	if output.Status != "planned" || len(output.Fields) != 14 || output.Fields[0].FieldName != "unique_key" {
		t.Fatalf("unexpected preview: %+v", output)
	}
}

func TestBootstrapReviewBaseCreatesResourcesAndSavesLocalEnv(t *testing.T) {
	const appToken = "bascn_secret_review_app"
	const tableID = "tbl_secret_review_items"
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodPost && r.URL.Path == "/bitable/v1/apps":
			var payload map[string]string
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode app create: %v", err)
			}
			if payload["name"] != defaultReviewBaseName || payload["time_zone"] != "Asia/Shanghai" {
				t.Fatalf("app payload = %+v", payload)
			}
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"app":{"app_token":"` + appToken + `","default_table_id":"tbl_default","name":"Review","url":"https://example.invalid/base"}}}`))
		case r.Method == http.MethodGet && r.URL.Path == "/bitable/v1/apps/"+appToken+"/tables":
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"items":[{"table_id":"tbl_default","name":"数据表"}],"has_more":false}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/bitable/v1/apps/"+appToken+"/tables":
			var payload struct {
				Table struct {
					Name   string             `json:"name"`
					Fields []BitableFieldSpec `json:"fields"`
				} `json:"table"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode table create: %v", err)
			}
			if payload.Table.Name != defaultReviewTableName || len(payload.Table.Fields) != 14 {
				t.Fatalf("table payload = %+v", payload.Table)
			}
			if payload.Table.Fields[0] != (BitableFieldSpec{FieldName: "unique_key", Type: 1}) {
				t.Fatalf("primary field = %+v", payload.Table.Fields[0])
			}
			assertReviewBootstrapField(t, payload.Table.Fields, "pr_number", 2)
			assertReviewBootstrapField(t, payload.Table.Fields, "archived", 7)
			_, _ = w.Write([]byte(`{"code":0,"msg":"ok","data":{"table_id":"` + tableID + `","default_view_id":"vew_review","field_id_list":["fld_1"]}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	envPath := filepath.Join(t.TempDir(), "feishu-review-resources.env.ps1")
	output, err := bootstrapReviewBase(context.Background(), ReviewBaseBootstrapOptions{
		AppID:     "cli_test",
		AppSecret: "secret",
		BaseName:  defaultReviewBaseName,
		TableName: defaultReviewTableName,
		ViewName:  defaultReviewViewName,
		TimeZone:  "Asia/Shanghai",
		OutputEnv: envPath,
		Send:      true,
	}, OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()})
	if err != nil {
		t.Fatalf("bootstrapReviewBase send: %v", err)
	}
	if requestCount != 4 || output.Status != "completed" || output.FeishuWrites != 2 {
		t.Fatalf("unexpected result: requests=%d output=%+v", requestCount, output)
	}
	rendered, _ := json.Marshal(output)
	if strings.Contains(string(rendered), appToken) || strings.Contains(string(rendered), tableID) {
		t.Fatalf("output leaked resource IDs: %s", rendered)
	}
	contents, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("read env: %v", err)
	}
	if !strings.Contains(string(contents), appToken) || !strings.Contains(string(contents), tableID) {
		t.Fatalf("resource env missing IDs: %s", contents)
	}
}

func TestBootstrapReviewBaseResumesExistingTableWithoutWrites(t *testing.T) {
	const appToken = "bascn_existing_review_app"
	const tableID = "tbl_existing_review_items"
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/auth/v3/tenant_access_token/internal":
			_, _ = w.Write([]byte(`{"code":0,"tenant_access_token":"tenant-token","expire":7200}`))
		case r.Method == http.MethodGet && r.URL.Path == "/bitable/v1/apps/"+appToken+"/tables":
			_, _ = w.Write([]byte(`{"code":0,"data":{"items":[{"table_id":"` + tableID + `","name":"Review WorkItems"}],"has_more":false}}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	envPath := filepath.Join(t.TempDir(), "resources.ps1")
	output, err := bootstrapReviewBase(context.Background(), ReviewBaseBootstrapOptions{
		AppID:          "cli_test",
		AppSecret:      "secret",
		BaseName:       defaultReviewBaseName,
		TableName:      defaultReviewTableName,
		ViewName:       defaultReviewViewName,
		ResumeAppToken: appToken,
		OutputEnv:      envPath,
		Send:           true,
	}, OpenAPIClient{BaseURL: server.URL, HTTP: server.Client()})
	if err != nil {
		t.Fatalf("bootstrapReviewBase resume: %v", err)
	}
	if requestCount != 2 || output.FeishuWrites != 0 || output.BaseAction != "reuse" || output.TableAction != "reused" {
		t.Fatalf("unexpected resume result: requests=%d output=%+v", requestCount, output)
	}
}

func assertReviewBootstrapField(t *testing.T, fields []BitableFieldSpec, name string, fieldType int) {
	t.Helper()
	for _, field := range fields {
		if field.FieldName == name {
			if field.Type != fieldType {
				t.Fatalf("field %s type = %d", name, field.Type)
			}
			return
		}
	}
	t.Fatalf("field %s not found", name)
}

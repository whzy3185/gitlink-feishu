package template

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestTemplateList(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/project_templates.json")
		writeJSON(t, w, map[string]interface{}{
			"total_count": 1,
			"project_templates": []interface{}{
				map[string]interface{}{
					"id":         float64(1),
					"type":       "ProjectTemplates::Issue",
					"name":       "bug修复模板",
					"content":    "## 问题描述\n[描述]",
					"created_at": "2026-01-13 17:30:20",
					"updated_at": "2026-01-13 17:30:20",
				},
			},
		})
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "list", map[string]string{})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestTemplateGet(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/project_templates/42.json")
		writeJSON(t, w, map[string]interface{}{
			"project_template": map[string]interface{}{
				"id":         float64(42),
				"type":       "ProjectTemplates::Issue",
				"name":       "功能请求模板",
				"content":    "## 功能描述\n[描述]",
				"created_at": "2026-01-22 11:15:04",
				"updated_at": "2026-01-22 11:15:04",
			},
		})
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "get", map[string]string{"id": "42"})
	if err != nil {
		t.Fatalf("get shortcut failed: %v", err)
	}
}

func TestTemplateGetRequiresID(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("get without id should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "get", map[string]string{})
	if err == nil {
		t.Fatal("expected get without id to return an error")
	}
}

func TestTemplateCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/v1/owner/repo/project_templates.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "create", map[string]string{
		"type":    "ProjectTemplates::Issue",
		"name":    "测试issue模板",
		"content": "模板内容",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, payload["type"], "ProjectTemplates::Issue")
	assertEqual(t, payload["name"], "测试issue模板")
	assertEqual(t, payload["content"], "模板内容")
}

func TestTemplateCreateRejectsInvalidType(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid type should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "create", map[string]string{
		"type":    "InvalidType",
		"name":    "test",
		"content": "content",
	})
	if err == nil {
		t.Fatal("expected invalid type to return an error")
	}
}

func TestTemplateUpdatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "PUT", "/v1/owner/repo/project_templates/5.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "update", map[string]string{
		"id":      "5",
		"type":    "ProjectTemplates::Issue",
		"name":    "更新模板",
		"content": "更新内容",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	assertEqual(t, payload["type"], "ProjectTemplates::Issue")
	assertEqual(t, payload["name"], "更新模板")
	assertEqual(t, payload["content"], "更新内容")
}

func TestTemplateDelete(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/owner/repo/project_templates/5.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "delete", map[string]string{"id": "5"})
	if err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestTemplateDeleteRequiresID(t *testing.T) {
	server := newTemplateTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("delete without id should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runTemplateShortcut(t, server, "delete", map[string]string{})
	if err == nil {
		t.Fatal("expected delete without id to return an error")
	}
}

func TestIsValidTemplateType(t *testing.T) {
	for _, valid := range templateTypes {
		if !isValidTemplateType(valid) {
			t.Fatalf("expected %q to be valid", valid)
		}
	}
	invalid := []string{"", "InvalidType", "projecttemplates::issue", "PR"}
	for _, v := range invalid {
		if isValidTemplateType(v) {
			t.Fatalf("expected %q to be invalid", v)
		}
	}
}

func TestTemplatePath(t *testing.T) {
	ctx := &common.RuntimeContext{Owner: "gitlink", Repo: "forgeplus"}
	got := templatePath(ctx)
	want := "/v1/gitlink/forgeplus/project_templates"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestTemplateItemPath(t *testing.T) {
	ctx := &common.RuntimeContext{Owner: "gitlink", Repo: "forgeplus"}
	got := templateItemPath(ctx, "42")
	want := "/v1/gitlink/forgeplus/project_templates/42"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

// --- helpers ---

func runTemplateShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findTemplateShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
		Tr:     i18n.Default(),
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findTemplateShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newTemplateTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

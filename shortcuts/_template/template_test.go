package template

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

// newTestContext 返回指向本地假服务的 RuntimeContext（惯例：单测不访问真实网络）。
func newTestContext(t *testing.T, handler http.HandlerFunc, args map[string]string) (*common.RuntimeContext, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "demo-owner",
		Repo:   "demo-repo",
		Format: "json",
		Args:   args,
	}, server
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func TestListSendsQueryParams(t *testing.T) {
	ctx, _ := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/demo-owner/demo-repo/gadgets.json" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.URL.Query().Get("keyword") != "abc" || r.URL.Query().Get("page") != "2" {
			t.Fatalf("query = %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"gadgets":[]}`))
	}, map[string]string{"keyword": "abc", "page": "2", "limit": "20"})

	if err := findShortcut(t, "list").Run(ctx); err != nil {
		t.Fatalf("list error: %v", err)
	}
}

func TestCreateRequiresName(t *testing.T) {
	ctx, _ := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected when required flag missing")
	}, map[string]string{})

	if err := findShortcut(t, "create").Run(ctx); err == nil {
		t.Fatal("expected error for missing --name")
	}
}

func TestDeleteRefusesWithoutYes(t *testing.T) {
	ctx, _ := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no request expected without --yes")
	}, map[string]string{"id": "7"})

	if err := findShortcut(t, "delete").Run(ctx); err == nil {
		t.Fatal("expected confirmation error without --yes")
	}
}

func TestDeleteSendsDelete(t *testing.T) {
	ctx, _ := newTestContext(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/v1/demo-owner/demo-repo/gadgets/7.json" {
			t.Fatalf("%s %s", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":0}`))
	}, map[string]string{"id": "7", "yes": "true"})

	if err := findShortcut(t, "delete").Run(ctx); err != nil {
		t.Fatalf("delete error: %v", err)
	}
}

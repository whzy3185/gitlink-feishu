package wiki

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestWikiPages(t *testing.T) {
	tests := []struct {
		name        string
		mockStatus  int
		mockBody    string
		wantErr     bool
		errContains string
	}{
		{"正常返回", 200, `{"wikiPages": []}`, false, ""},
		{"API 404", 404, `{"error": "not found"}`, true, "404"},
		{"返回 HTML", 200, `<!DOCTYPE html><html><body>Login</body></html>`, true, "HTML"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("expected GET, got %s", r.Method)
				}
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			shortcut := findWikiShortcut(t, "pages")
			ctx := &common.RuntimeContext{
				Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
				Owner:  "test", Repo: "test", Format: "json",
				Args: map[string]string{},
			}
			err := shortcut.Run(ctx)

			if tt.wantErr && err == nil {
				t.Fatal("期望错误但为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("不期望错误: %v", err)
			}
			if tt.wantErr && tt.errContains != "" && err != nil {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("错误应包含 %q: %s", tt.errContains, err.Error())
				}
			}
		})
	}
}

func TestWikiGet(t *testing.T) {
	tests := []struct {
		name        string
		args        map[string]string
		mockStatus  int
		mockBody    string
		wantErr     bool
		errContains string
	}{
		{"正常获取", map[string]string{"id": "42"}, 200, `{"id": 42, "title": "Home"}`, false, ""},
		{"缺少 id", map[string]string{}, 200, `{}`, true, ""},
		{"API 404", map[string]string{"id": "999"}, 404, `{"error": "not found"}`, true, "404"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.Contains(r.URL.Path, "/api/wiki/getWiki") {
					t.Errorf("expected path containing /api/wiki/getWiki, got %s", r.URL.Path)
				}
				w.WriteHeader(tt.mockStatus)
				w.Write([]byte(tt.mockBody))
			}))
			defer server.Close()

			shortcut := findWikiShortcut(t, "get")
			ctx := &common.RuntimeContext{
				Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
				Owner:  "test", Repo: "test", Format: "json",
				Args: tt.args,
			}
			err := shortcut.Run(ctx)

			if tt.wantErr && err == nil {
				t.Fatal("期望错误但为 nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("不期望错误: %v", err)
			}
		})
	}
}

func TestWikiCreate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/wiki/createWiki") {
			t.Errorf("expected path containing /api/wiki/createWiki, got %s", r.URL.Path)
		}
		payload = decodeWikiJSON(t, r)
		w.WriteHeader(200)
		w.Write([]byte(`{"status": 0, "message": "success"}`))
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "create")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"title":   "Getting Started",
			"content": "# Hello\nWelcome to the wiki",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
	if payload["title"] != "Getting Started" {
		t.Errorf("expected title 'Getting Started', got %v", payload["title"])
	}
	if payload["content"] != "# Hello\nWelcome to the wiki" {
		t.Errorf("unexpected content: %v", payload["content"])
	}
}

func TestWikiCreateWithProject(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		payload = decodeWikiJSON(t, r)
		w.WriteHeader(200)
		w.Write([]byte(`{"status": 0}`))
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "create")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"title":   "Test",
			"content": "Body",
			"project": "123",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
	if payload["project_id"] != "123" {
		t.Errorf("expected project_id '123', got %v", payload["project_id"])
	}
}

func TestWikiCreateMissingTitle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API when --title is missing")
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "create")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"content": "only content"},
	}
	if err := shortcut.Run(ctx); err == nil {
		t.Fatal("expected error when --title is missing")
	}
}

func TestWikiUpdate(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/wiki/updateWiki") {
			t.Errorf("expected path containing /api/wiki/updateWiki, got %s", r.URL.Path)
		}
		payload = decodeWikiJSON(t, r)
		w.WriteHeader(200)
		w.Write([]byte(`{"status": 0, "message": "success"}`))
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "update")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"id":      "42",
			"title":   "Updated Title",
			"content": "Updated content",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}
	if payload["id"] != "42" {
		t.Errorf("expected id '42', got %v", payload["id"])
	}
	if payload["title"] != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %v", payload["title"])
	}
}

func TestWikiDelete(t *testing.T) {
	var payload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/api/wiki/deleteWiki") {
			t.Errorf("expected path containing /api/wiki/deleteWiki, got %s", r.URL.Path)
		}
		payload = decodeWikiJSON(t, r)
		w.WriteHeader(200)
		w.Write([]byte(`{"status": 0, "message": "success"}`))
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "delete")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{"id": "42"},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
	if payload["id"] != "42" {
		t.Errorf("expected id '42', got %v", payload["id"])
	}
}

func TestWikiDeleteMissingId(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not call API when --id is missing")
	}))
	defer server.Close()

	shortcut := findWikiShortcut(t, "delete")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{},
	}
	if err := shortcut.Run(ctx); err == nil {
		t.Fatal("expected error when --id is missing")
	}
}

func findWikiShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func decodeWikiJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestAPIError(t *testing.T) {
	err := &APIError{StatusCode: 404, Code: "not_found", Message: "PR not found"}
	if err.Error() != "[not_found] PR not found" {
		t.Fatalf("Error() = %q, want %q", err.Error(), "[not_found] PR not found")
	}
}

func TestSuggestFix(t *testing.T) {
	tests := []struct {
		code int
		want string
	}{
		{401, "请先运行 gitlink-cli auth login 登录"},
		{403, "权限不足，请确认账户权限或联系项目管理员"},
		{404, "资源不存在，请检查 owner/repo/id 是否正确"},
		{422, "参数校验失败，请检查请求参数"},
		{500, ""},
		{0, ""},
	}
	for _, tt := range tests {
		t.Run(http.StatusText(tt.code), func(t *testing.T) {
			if got := suggestFix(tt.code); got != tt.want {
				t.Fatalf("suggestFix(%d) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestClientDoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":{"key":"value"}}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.OK {
		t.Fatal("expected OK=true")
	}
}

func TestClientDoJSONSuffix(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test.json" {
			t.Fatalf("expected path /api/test.json, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoJSONSuffixPreserved(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/test.json" {
			t.Fatalf("expected path /api/test.json, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Do("GET", "/api/test.json", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "open" {
			t.Fatalf("expected state=open, got %s", r.URL.Query().Get("state"))
		}
		if r.URL.Query().Get("page") != "1" {
			t.Fatalf("expected page=1, got %s", r.URL.Query().Get("page"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	q := url.Values{}
	q.Set("state", "open")
	q.Set("page", "1")
	_, err := c.Do("GET", "/api/test", nil, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Do("GET", "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 404 {
		t.Fatalf("StatusCode = %d, want 404", apiErr.StatusCode)
	}
}

func TestClientDoNonJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("plain text response"))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.OK {
		t.Fatal("expected OK=true for non-JSON response")
	}
}

func TestClientDoStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":403,"message":"Forbidden"}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("GET", "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for status=403")
	}
	if env == nil {
		t.Fatal("expected envelope for status error")
	}
	if env.OK {
		t.Fatal("expected OK=false for status error")
	}
}

func TestClientDoStatusZero(t *testing.T) {
	// status=0, 200, 1 are treated as success
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":0,"data":"ok"}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.OK {
		t.Fatal("expected OK=true for status=0")
	}
}

func TestClientDoPaginationMeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"total_count":100,"page":1,"limit":20,"data":[]}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env.Meta == nil {
		t.Fatal("expected Meta to be populated")
	}
	if env.Meta.TotalCount != 100 {
		t.Fatalf("TotalCount = %d, want 100", env.Meta.TotalCount)
	}
	if env.Meta.Page != 1 {
		t.Fatalf("Page = %d, want 1", env.Meta.Page)
	}
	if env.Meta.Limit != 20 {
		t.Fatalf("Limit = %d, want 20", env.Meta.Limit)
	}
}

func TestClientDoPathWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The path query param should be preserved
		if r.URL.Query().Get("filepath") != "test.go" {
			t.Fatalf("expected filepath=test.go, got %s", r.URL.Query().Get("filepath"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Do("GET", "/api/sub_entries?filepath=test.go", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoPathWithQueryAndExtraParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("ref") != "master" {
			t.Fatalf("expected ref=master, got %s", r.URL.Query().Get("ref"))
		}
		if r.URL.Query().Get("filepath") != "test.go" {
			t.Fatalf("expected filepath=test.go, got %s", r.URL.Query().Get("filepath"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	q := url.Values{}
	q.Set("ref", "master")
	_, err := c.Do("GET", "/api/sub_entries?filepath=test.go", nil, q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoWithBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":123}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	env, err := c.Do("POST", "/api/create", map[string]string{"title": "test"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !env.OK {
		t.Fatal("expected OK=true")
	}
}

func TestClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Get("/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Post("/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientPut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Fatalf("expected PUT, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Put("/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Delete("/api/test", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoInvalidURL(t *testing.T) {
	c := &Client{HTTP: &http.Client{}, BaseURL: "://invalid"}
	_, err := c.Do("GET", "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestClientDebug(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL, Debug: true}
	_, err := c.Do("GET", "/api/test", nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoStatusInt(t *testing.T) {
	// Some APIs return status as int, not float64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":404,"message":"Not Found"}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.Do("GET", "/api/test", nil, nil)
	if err == nil {
		t.Fatal("expected error for status=404 (int)")
	}
}

func TestClientNew(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GITLINK_CONFIG_DIR", dir)
	cfgPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(cfgPath, []byte("base_url: https://gitlink.example.com/api/v1\n"), 0644)

	cli, err := New()
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if cli.BaseURL != "https://gitlink.example.com/api/v1" {
		t.Fatalf("BaseURL = %q", cli.BaseURL)
	}
	if cli.HTTP == nil {
		t.Fatal("HTTP client is nil")
	}
}

func TestPaginateAllSinglePage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":1},{"id":2}]}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/test", nil)
	if err != nil {
		t.Fatalf("PaginateAll error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestPaginateAllMultiPage(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		page := r.URL.Query().Get("page")
		if page == "1" {
			w.Write([]byte(`{"data":[{"id":1},{"id":2}]}`))
		} else {
			w.Write([]byte(`{"data":[{"id":3}]}`))
		}
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	params := url.Values{}
	params.Set("limit", "2")
	items, err := c.PaginateAll("/test", params)
	if err != nil {
		t.Fatalf("PaginateAll error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}

func TestPaginateAllWrappedData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":1},{"id":2}],"total_count":2}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/test", nil)
	if err != nil {
		t.Fatalf("PaginateAll error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestPaginateAllSingleObject(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name":"single-object"}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	items, err := c.PaginateAll("/test", nil)
	if err != nil {
		t.Fatalf("PaginateAll error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	var data map[string]interface{}
	json.Unmarshal(items[0], &data)
	if data["name"] != "single-object" {
		t.Fatalf("unexpected data: %v", data)
	}
}

func TestPaginateAllHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.PaginateAll("/test", nil)
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestPaginateAllNotOK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":500,"message":"error"}`))
	}))
	defer server.Close()

	c := &Client{HTTP: server.Client(), BaseURL: server.URL}
	_, err := c.PaginateAll("/test", nil)
	if err == nil {
		t.Fatal("expected error when envelope ok=false")
	}
}

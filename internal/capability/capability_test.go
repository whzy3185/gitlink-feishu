package capability

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/client"
)

// newTestRegistry creates a Registry with a temp cache file to avoid
// interference from real CLI cache files.
func newTestRegistry(t *testing.T) *Registry {
	t.Helper()
	r := &Registry{
		statuses:  make(map[string]*DomainStatus),
		cachePath: filepath.Join(t.TempDir(), "capabilities.json"),
	}
	return r
}

func TestStatusString(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusUnknown, "unknown"},
		{StatusAvailable, "available"},
		{StatusUnavailable, "unavailable"},
		{StatusError, "error"},
	}
	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusUnknown, "?"},
		{StatusAvailable, "✓"},
		{StatusUnavailable, "⚠"},
		{StatusError, "✗"},
	}
	for _, tt := range tests {
		if got := tt.status.Emoji(); got != tt.want {
			t.Errorf("Status(%d).Emoji() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestProbeOneAvailable(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":[]}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/test", NeedsRepo: false}

	ds := r.probeOne(cli, "test", canary, "", "")
	if ds.Status != StatusAvailable {
		t.Errorf("expected StatusAvailable, got %s", ds.Status)
	}
	if ds.Message != "API 正常响应" {
		t.Errorf("Message = %q, want 'API 正常响应'", ds.Message)
	}
}

func TestProbeOneHTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html><html><body>Login</body></html>`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/test", NeedsRepo: false}

	ds := r.probeOne(cli, "test", canary, "", "")
	if ds.Status != StatusUnavailable {
		t.Errorf("expected StatusUnavailable, got %s", ds.Status)
	}
	if !strings.Contains(ds.Message, "尚未实现") {
		t.Errorf("Message should mention '未实现', got %q", ds.Message)
	}
}

func TestProbeOne404(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/test", NeedsRepo: false}

	ds := r.probeOne(cli, "test", canary, "", "")
	if ds.Status != StatusUnavailable {
		t.Errorf("expected StatusUnavailable, got %s", ds.Status)
	}
	if !strings.Contains(ds.Message, "404") {
		t.Errorf("Message should mention 404, got %q", ds.Message)
	}
}

func TestProbeOne401(t *testing.T) {
	// 401 means endpoint exists but auth is needed — should be Available
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"status":401,"message":"请登录后再操作"}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/test", NeedsRepo: false}

	ds := r.probeOne(cli, "test", canary, "", "")
	if ds.Status != StatusAvailable {
		t.Errorf("expected StatusAvailable for 401 (endpoint exists), got %s", ds.Status)
	}
}

func TestProbeOne403(t *testing.T) {
	// 403 means endpoint exists but permission denied — should be Available
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"status":403,"message":"您没有权限进行该操作"}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/test", NeedsRepo: false}

	ds := r.probeOne(cli, "test", canary, "", "")
	if ds.Status != StatusAvailable {
		t.Errorf("expected StatusAvailable for 403 (endpoint exists), got %s", ds.Status)
	}
}

func TestProbeOneWithRepoPlaceholders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/myowner/myrepo/issue_tags.json" {
			t.Errorf("path = %s, want /api/v1/myowner/myrepo/issue_tags.json", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":[]}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	canary := CanaryProbe{Method: "GET", Path: "/api/v1/{owner}/{repo}/issue_tags", NeedsRepo: true}

	ds := r.probeOne(cli, "label", canary, "myowner", "myrepo")
	if ds.Status != StatusAvailable {
		t.Errorf("expected StatusAvailable, got %s", ds.Status)
	}
}

func TestProbeAllSkipsRepoProbesWhenNoContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}

	// Probe with empty owner/repo — repo-dependent probes should be skipped
	results := r.ProbeAll(cli, "", "")

	// Repo-less endpoints (notification, pm, wiki, pipeline, search) should be probed
	for _, domain := range []string{"notification", "pm", "pipeline", "search"} {
		if _, ok := results[domain]; !ok {
			t.Errorf("domain %q should be probed (no repo needed), but was skipped", domain)
		}
	}

	// Repo-dependent endpoints should be skipped
	for _, domain := range []string{"label", "webhook", "member", "milestone", "export", "wiki", "workflow"} {
		if _, ok := results[domain]; ok {
			t.Errorf("domain %q needs repo context, should be skipped, but was probed", domain)
		}
	}
}

func TestProbeAllWithRepoContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":[]}`))
	}))
	defer server.Close()

	r := newTestRegistry(t)
	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}

	results := r.ProbeAll(cli, "owner", "repo")

	// All domains should be probed when repo context is available
	allDomains := []string{
		"label", "notification", "pm", "wiki", "pipeline",
		"webhook", "member", "milestone", "export", "search", "workflow",
	}
	for _, domain := range allDomains {
		if _, ok := results[domain]; !ok {
			t.Errorf("domain %q should be probed, but was skipped", domain)
		}
	}
}

func TestGetSet(t *testing.T) {
	r := newTestRegistry(t)

	// Initial state: unknown
	if r.Get("label") != StatusUnknown {
		t.Error("expected StatusUnknown before any probe")
	}

	// Manually set a status via ProbeAll
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	cli := &client.Client{HTTP: server.Client(), BaseURL: server.URL}
	r.ProbeAll(cli, "", "")

	// After probe, notification should be known (repo-less endpoint was probed)
	if r.Get("notification") == StatusUnknown {
		t.Error("notification should have been probed")
	}
}

func TestIsStale(t *testing.T) {
	r := newTestRegistry(t)
	if !r.IsStale() {
		t.Error("empty registry should be stale")
	}

	// Add a fresh entry
	r.mu.Lock()
	r.statuses["test"] = &DomainStatus{
		Domain:      "test",
		Status:      StatusAvailable,
		LastChecked: time.Now(),
	}
	r.mu.Unlock()

	if r.IsStale() {
		t.Error("registry with fresh entry should not be stale")
	}

	// Add a stale entry
	r.mu.Lock()
	r.statuses["stale"] = &DomainStatus{
		Domain:      "stale",
		Status:      StatusAvailable,
		LastChecked: time.Now().Add(-48 * time.Hour),
	}
	r.mu.Unlock()

	if !r.IsStale() {
		t.Error("registry with stale entry should be stale")
	}
}

func TestGetAll(t *testing.T) {
	r := newTestRegistry(t)
	r.mu.Lock()
	r.statuses["test"] = &DomainStatus{Domain: "test", Status: StatusAvailable}
	r.mu.Unlock()

	all := r.GetAll()
	if len(all) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(all))
	}
	if all["test"].Status != StatusAvailable {
		t.Error("GetAll should return a copy with correct data")
	}
}

func TestCacheSaveLoad(t *testing.T) {
	dir := t.TempDir()

	r := newTestRegistry(t)
	r.cachePath = filepath.Join(dir, "capabilities.json")

	// Add some data
	r.mu.Lock()
	r.statuses["test"] = &DomainStatus{
		Domain:      "test",
		Status:      StatusAvailable,
		Message:     "working",
		LastChecked: time.Now(),
	}
	r.mu.Unlock()

	// Save
	r.save()
	if _, err := os.Stat(r.cachePath); os.IsNotExist(err) {
		t.Fatal("cache file was not created")
	}

	// Load into new registry (no load from disk — just read the saved file)
	r2 := &Registry{
		statuses:  make(map[string]*DomainStatus),
		cachePath: r.cachePath,
	}
	r2.load()

	if r2.Get("test") != StatusAvailable {
		t.Errorf("loaded status = %s, want available", r2.Get("test"))
	}
}

func TestCacheFileRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "capabilities.json")

	// Create and populate registry
	r := newTestRegistry(t)
	r.cachePath = cachePath
	r.mu.Lock()
	r.statuses["search"] = &DomainStatus{
		Domain:      "search",
		Status:      StatusUnavailable,
		Message:     "API 端点不存在（404）",
		LastChecked: time.Now(),
	}
	r.mu.Unlock()
	r.save()

	// Verify JSON structure
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]*DomainStatus
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["search"].Status != StatusUnavailable {
		t.Errorf("decoded status = %d, want %d", decoded["search"].Status, StatusUnavailable)
	}
}

func TestSummary(t *testing.T) {
	r := newTestRegistry(t)
	r.mu.Lock()
	// Use domains from the hardcoded list in Summary()
	r.statuses["search"] = &DomainStatus{Domain: "search", Status: StatusAvailable, Message: "API 正常响应"}
	r.statuses["wiki"] = &DomainStatus{Domain: "wiki", Status: StatusUnavailable, Message: "后端 API 尚未实现该端点"}
	r.mu.Unlock()

	summary := r.Summary()
	if !strings.Contains(summary, "search") {
		t.Error("Summary should contain domain name 'search'")
	}
	if !strings.Contains(summary, "wiki") {
		t.Error("Summary should contain domain name 'wiki'")
	}
	if !strings.Contains(summary, "available") {
		t.Error("Summary should contain status text")
	}
}

func TestCanaryEndpointsHaveValidDomains(t *testing.T) {
	// Verify that all canary endpoints map to known domains
	for domain, canary := range canaryEndpoints {
		if canary.Method == "" {
			t.Errorf("domain %q: Method is empty", domain)
		}
		if canary.Path == "" {
			t.Errorf("domain %q: Path is empty", domain)
		}
		if canary.NeedsRepo && !strings.Contains(canary.Path, "{owner}") && !strings.Contains(canary.Path, "{repo}") {
			t.Errorf("domain %q: NeedsRepo=true but path has no owner/repo placeholder", domain)
		}
	}
}

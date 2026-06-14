package dataset

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "alice",
		Repo:   "demo",
		Format: "json",
		Args:   args,
	}
	return shortcut.Run(ctx)
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

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// --- list ---

func TestDatasetListNormalizesIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/project_datasets.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "1,2,3" {
			t.Fatalf("ids = %q, want 1,2,3", got)
		}
		writeJSON(w, map[string]interface{}{"total_count": float64(0), "project_datasets": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"ids": " 1, 2 ,3 "}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestDatasetListMissingIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected when ids missing")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{}); err == nil {
		t.Fatal("expected error for missing --ids")
	}
}

func TestDatasetListInvalidIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid ids")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"ids": "abc"}); err == nil {
		t.Fatal("expected error for non-numeric --ids")
	}
}

// --- view ---

func TestDatasetViewResolvesProjectID(t *testing.T) {
	var sawRepo, sawQuery bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/alice/demo.json":
			sawRepo = true
			writeJSON(w, map[string]interface{}{"id": float64(5988)})
		case "/v1/project_datasets.json":
			sawQuery = true
			if got := r.URL.Query().Get("ids"); got != "5988" {
				t.Fatalf("ids = %q, want 5988", got)
			}
			writeJSON(w, map[string]interface{}{"total_count": float64(1), "project_datasets": []interface{}{}})
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
	if !sawRepo || !sawQuery {
		t.Fatalf("expected repo+query calls, got repo=%v query=%v", sawRepo, sawQuery)
	}
}

func TestDatasetViewExplicitProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/project_datasets.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("ids"); got != "42" {
			t.Fatalf("ids = %q, want 42", got)
		}
		writeJSON(w, map[string]interface{}{"total_count": float64(0), "project_datasets": []interface{}{}})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{"project-id": "42"}); err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestDatasetViewInvalidProjectID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for invalid project id")
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{"project-id": "x"}); err == nil {
		t.Fatal("expected error for invalid --project-id")
	}
}

func TestDatasetViewHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "view", map[string]string{"project-id": "42"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

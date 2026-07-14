package star

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestStar(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(100)})
		case r.Method == "POST" && r.URL.Path == "/projects/100/praise_tread/like.json":
			writeJSON(t, w, map[string]interface{}{})
		default:
			t.Fatalf("unexpected request #%d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runStarShortcut(t, server, "star", map[string]string{}); err != nil {
		t.Fatalf("star failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("expected 2 API calls, got %d", callCount)
	}
}

func TestUnstar(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == "GET" && r.URL.Path == "/owner/repo.json":
			writeJSON(t, w, map[string]interface{}{"id": float64(100)})
		case r.Method == "DELETE" && r.URL.Path == "/projects/100/praise_tread/unlike.json":
			writeJSON(t, w, map[string]interface{}{})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	if err := runStarShortcut(t, server, "unstar", map[string]string{}); err != nil {
		t.Fatalf("unstar failed: %v", err)
	}
}

func TestStars(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/owner/repo/stargazers.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{
			"count": 1,
			"users": []map[string]interface{}{{"login": "alice"}},
		})
	}))
	defer server.Close()

	err := runStarShortcut(t, server, "stars", map[string]string{
		"owner": "owner", "repo": "repo",
	})
	if err != nil {
		t.Fatalf("stars failed: %v", err)
	}
}

func runStarShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findStarShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner", Repo: "repo", Format: "json", Args: args,
	}
	return shortcut.Run(ctx)
}

func findStarShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(payload)
}

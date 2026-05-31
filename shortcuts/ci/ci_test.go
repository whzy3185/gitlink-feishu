package ci

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
		Owner:  "owner",
		Repo:   "repo",
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
	json.NewEncoder(w).Encode(v)
}

// --- builds ---

func TestCIBuilds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/builds.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"number": float64(1), "status": "success"},
		})
	}))
	defer server.Close()

	err := runShortcut(t, server, "builds", map[string]string{"page": "1", "limit": "20"})
	if err != nil {
		t.Fatalf("builds failed: %v", err)
	}
}

// --- logs ---

func TestCILogs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/builds/5/logs/1/1.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"log": "Build output..."})
	}))
	defer server.Close()

	err := runShortcut(t, server, "logs", map[string]string{"build": "5", "stage": "1", "step": "1"})
	if err != nil {
		t.Fatalf("logs failed: %v", err)
	}
}

func TestCILogsDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/builds/3/logs/1/1.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"log": "output"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "logs", map[string]string{"build": "3"})
	if err != nil {
		t.Fatalf("logs with defaults failed: %v", err)
	}
}

// --- restart ---

func TestCIRestart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/builds/7/restart.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "restarted"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "restart", map[string]string{"build": "7"})
	if err != nil {
		t.Fatalf("restart failed: %v", err)
	}
}

// --- stop ---

func TestCIStop(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/owner/repo/builds/7/stop.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"message": "stopped"})
	}))
	defer server.Close()

	err := runShortcut(t, server, "stop", map[string]string{"build": "7"})
	if err != nil {
		t.Fatalf("stop failed: %v", err)
	}
}

// --- HTTP error paths ---

func TestCIBuildsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "builds", map[string]string{"page": "1", "limit": "20"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestCILogsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "logs", map[string]string{"build": "5"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestCIRestartHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "restart", map[string]string{"build": "7"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestCIStopHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	err := runShortcut(t, server, "stop", map[string]string{"build": "7"})
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

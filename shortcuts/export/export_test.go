package export

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestExportIssues(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`[{"id":1,"subject":"Bug fix","status":1,"created_at":"2026-01-01"}]`))
	}))
	defer server.Close()

	shortcut := findExportShortcut(t, "issues")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"format": "json",
			"output": os.TempDir() + "/test_issues_export.json",
			"state":  "all",
			"page":   "1",
			"limit":  "50",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("export issues failed: %v", err)
	}
}

func TestExportPrs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`[{"id":2,"title":"Feature PR","status":0,"created_at":"2026-02-01"}]`))
	}))
	defer server.Close()

	shortcut := findExportShortcut(t, "prs")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"format": "json",
			"output": os.TempDir() + "/test_prs_export.json",
			"state":  "all",
			"page":   "1",
			"limit":  "50",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("export prs failed: %v", err)
	}
}

func TestExportContributors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.WriteHeader(200)
		w.Write([]byte(`[{"id":1,"login":"dev1","contributions":42}]`))
	}))
	defer server.Close()

	shortcut := findExportShortcut(t, "contributors")
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "test", Repo: "test", Format: "json",
		Args: map[string]string{
			"format": "json",
			"output": os.TempDir() + "/test_contributors_export.json",
		},
	}
	if err := shortcut.Run(ctx); err != nil {
		t.Fatalf("export contributors failed: %v", err)
	}
}

func TestExportUnsupportedFormat(t *testing.T) {
	items := []json.RawMessage{[]byte(`{"id":1}`)}
	err := writeExport("xml", "/dev/null", items, issueCSVHeader, issueCSVRow)
	if err == nil {
		t.Fatal("expected error for unsupported format")
	}
	if !strings.Contains(err.Error(), "不支持的格式") {
		t.Errorf("error should mention unsupported format: %v", err)
	}
}

func TestWriteCSV(t *testing.T) {
	tmpFile := os.TempDir() + "/test_export_write.csv"
	defer os.Remove(tmpFile)

	items := []json.RawMessage{
		[]byte(`{"id":1,"subject":"First","status":1,"created_at":"2026-01-01"}`),
		[]byte(`{"id":2,"subject":"Second","status":0,"created_at":"2026-01-02"}`),
	}
	if err := writeCSV(tmpFile, items, issueCSVHeader, issueCSVRow); err != nil {
		t.Fatalf("writeCSV failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "id,title,state,created_at") {
		t.Errorf("CSV header missing in output: %s", content)
	}
	if !strings.Contains(content, "First") {
		t.Errorf("expected 'First' in CSV output: %s", content)
	}
}

func TestWriteJSON(t *testing.T) {
	tmpFile := os.TempDir() + "/test_export_write.json"
	defer os.Remove(tmpFile)

	items := []json.RawMessage{
		[]byte(`{"id":1,"name":"test"}`),
	}
	if err := writeJSON(tmpFile, items); err != nil {
		t.Fatalf("writeJSON failed: %v", err)
	}

	data, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}
	if !strings.Contains(string(data), `"id": 1`) && !strings.Contains(string(data), `"id":1`) {
		t.Errorf("JSON content unexpected: %s", string(data))
	}
}

func findExportShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

package issue

import (
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseIssueNumbers(t *testing.T) {
	got, err := parseIssueNumbers("1, 2,2, 3")
	if err != nil {
		t.Fatalf("parseIssueNumbers returned error: %v", err)
	}
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseIssueNumbers() = %#v, want %#v", got, want)
	}
}

func TestParseIssueNumbersRejectsInvalidNumber(t *testing.T) {
	if _, err := parseIssueNumbers("1,abc"); err == nil {
		t.Fatal("parseIssueNumbers() expected an error for a non-integer issue number")
	}
}

func TestReadIssueNumbersFromCSVWithHeader(t *testing.T) {
	path := writeTempCSV(t, "title,number,state\nfirst,12,open\nsecond,13,open\n")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV returned error: %v", err)
	}
	want := []string{"12", "13"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIssueNumbersFromCSV() = %#v, want %#v", got, want)
	}
}

func TestReadIssueNumbersFromCSVWithProjectIssuesIndexHeader(t *testing.T) {
	path := writeTempCSV(t, "title,project_issues_index,state\nfirst,12,open\nsecond,13,open\n")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV returned error: %v", err)
	}
	want := []string{"12", "13"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIssueNumbersFromCSV() = %#v, want %#v", got, want)
	}
}

func TestReadIssueNumbersFromCSVWithoutHeaderUsesFirstColumn(t *testing.T) {
	path := writeTempCSV(t, "21,open\n22,closed\n21,duplicate\n")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV returned error: %v", err)
	}
	want := []string{"21", "22"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIssueNumbersFromCSV() = %#v, want %#v", got, want)
	}
}

func TestCollectIssueNumbersMergesCLIAndCSV(t *testing.T) {
	path := writeTempCSV(t, "number\n2\n3\n")
	got, err := collectIssueNumbers("1,2", path)
	if err != nil {
		t.Fatalf("collectIssueNumbers returned error: %v", err)
	}
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectIssueNumbers() = %#v, want %#v", got, want)
	}
}

func TestParseBool(t *testing.T) {
	if !parseBool("true") {
		t.Fatal("parseBool(true) = false, want true")
	}
	if parseBool("") {
		t.Fatal("parseBool(empty) = true, want false")
	}
}

func TestParseIssueNumbersEmpty(t *testing.T) {
	got, err := parseIssueNumbers("")
	if err != nil {
		t.Fatalf("parseIssueNumbers returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("parseIssueNumbers() = %#v, want nil", got)
	}
}

func TestReadIssueNumbersFromCSVMissingFile(t *testing.T) {
	_, err := readIssueNumbersFromCSV("/nonexistent/file.csv")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadIssueNumbersFromCSVEmpty(t *testing.T) {
	path := writeTempCSV(t, "")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty CSV, got %#v", got)
	}
}

func TestNormalizeIssueNumbersEmpty(t *testing.T) {
	got, err := normalizeIssueNumbers([]string{"", " ", "  "})
	if err != nil {
		t.Fatalf("normalizeIssueNumbers error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
}

func TestMergeIssueNumbers(t *testing.T) {
	got := mergeIssueNumbers([]string{"1", "2"}, []string{"2", "3"}, nil)
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mergeIssueNumbers() = %#v, want %#v", got, want)
	}
}

func TestCollectIssueNumbersCSVOnly(t *testing.T) {
	path := writeTempCSV(t, "number\n5\n6\n")
	got, err := collectIssueNumbers("", path)
	if err != nil {
		t.Fatalf("collectIssueNumbers error: %v", err)
	}
	want := []string{"5", "6"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectIssueNumbers() = %#v, want %#v", got, want)
	}
}

func TestCollectIssueNumbersCLIOnly(t *testing.T) {
	got, err := collectIssueNumbers("1,2,3", "")
	if err != nil {
		t.Fatalf("collectIssueNumbers error: %v", err)
	}
	want := []string{"1", "2", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectIssueNumbers() = %#v, want %#v", got, want)
	}
}

func TestNormalizeIssueNumbersRejectsNonInt(t *testing.T) {
	_, err := normalizeIssueNumbers([]string{"abc"})
	if err == nil {
		t.Fatal("expected error for non-integer")
	}
}

func TestReadIssueNumbersFromCSVShortRow(t *testing.T) {
	// Number column is index 1; short row skips due to len check
	path := writeTempCSV(t, "title,number\nfirst,1\nsecond,\nthird,3\n")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV error: %v", err)
	}
	want := []string{"1", "3"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIssueNumbersFromCSV() = %#v, want %#v", got, want)
	}
}

func TestReadIssueNumbersFromCSVIssueNumberHeader(t *testing.T) {
	path := writeTempCSV(t, "issue_number,title\n42,test\n")
	got, err := readIssueNumbersFromCSV(path)
	if err != nil {
		t.Fatalf("readIssueNumbersFromCSV error: %v", err)
	}
	want := []string{"42"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("readIssueNumbersFromCSV() = %#v, want %#v", got, want)
	}
}

func TestParseBoolFalse(t *testing.T) {
	if parseBool("false") {
		t.Fatal("parseBool(false) = true, want false")
	}
	if parseBool("  FALSE  ") {
		t.Fatal("parseBool(FALSE) = true, want false")
	}
}

func TestCollectIssueNumbersInvalidCLI(t *testing.T) {
	_, err := collectIssueNumbers("abc,def", "")
	if err == nil {
		t.Fatal("expected error for invalid issue numbers")
	}
}

func TestCollectIssueNumbersCSVReadError(t *testing.T) {
	_, err := collectIssueNumbers("", "/nonexistent/file.csv")
	if err == nil {
		t.Fatal("expected error for missing CSV file")
	}
}

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "issues.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

func TestBatchUpdateDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-update", map[string]string{
		"ids":          "101,102",
		"status-id":    "3",
		"priority-id":  "2",
		"tag-ids":      "7,8",
		"assigner-ids": "11",
		"dry-run":      "true",
	})
	if err != nil {
		t.Fatalf("batch-update dry-run failed: %v", err)
	}
}

func TestBatchUpdateCallsAPI(t *testing.T) {
	var payload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" || r.URL.Path != "/v1/owner/repo/issues/batch_update.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runShortcut(t, server, "batch-update", map[string]string{
		"ids":          "101,102,101",
		"status-id":    "3",
		"priority-id":  "2",
		"milestone-id": "9",
		"tag-ids":      "7,8",
		"assigner-ids": "11,12",
	})
	if err != nil {
		t.Fatalf("batch-update failed: %v", err)
	}
	assertFloatSlice(t, payload["ids"], []float64{101, 102})
	assertEqual(t, payload["status_id"], float64(3))
	assertEqual(t, payload["priority_id"], float64(2))
	assertEqual(t, payload["milestone_id"], float64(9))
	assertFloatSlice(t, payload["issue_tag_ids"], []float64{7, 8})
	assertFloatSlice(t, payload["assigner_ids"], []float64{11, 12})
}

func TestBatchUpdateRequiresUpdateField(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected API call: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	if err := runShortcut(t, server, "batch-update", map[string]string{"ids": "101"}); err == nil {
		t.Fatal("expected error when no update fields are provided")
	}
}

func TestBatchUpdateRejectsInvalidIDs(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected API call: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []map[string]string{
		{"ids": "abc", "status-id": "3"},
		{"ids": "101", "status-id": "bad"},
		{"ids": "101", "tag-ids": "7,,8"},
	}
	for _, args := range cases {
		if err := runShortcut(t, server, "batch-update", args); err == nil {
			t.Fatalf("expected validation error for args %#v", args)
		}
	}
}

func TestBatchDeleteDryRun(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	if err := runShortcut(t, server, "batch-delete", map[string]string{"ids": "101,102", "dry-run": "true"}); err != nil {
		t.Fatalf("batch-delete dry-run failed: %v", err)
	}
}

func TestBatchDeleteRequiresYes(t *testing.T) {
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected API call without --yes: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	if err := runShortcut(t, server, "batch-delete", map[string]string{"ids": "101"}); err == nil {
		t.Fatal("expected --yes confirmation error")
	}
}

func TestBatchDeleteCallsAPIWithYes(t *testing.T) {
	var payload map[string]interface{}
	server := newIssueTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" || r.URL.Path != "/v1/owner/repo/issues/batch_destroy.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runShortcut(t, server, "batch-delete", map[string]string{"ids": "101,102,101", "yes": "true"}); err != nil {
		t.Fatalf("batch-delete failed: %v", err)
	}
	assertFloatSlice(t, payload["ids"], []float64{101, 102})
}

func assertFloatSlice(t *testing.T, got interface{}, want []float64) {
	t.Helper()
	items, ok := got.([]interface{})
	if !ok {
		t.Fatalf("got %#v, want []interface{}", got)
	}
	if len(items) != len(want) {
		t.Fatalf("got len %d, want %d: %#v", len(items), len(want), got)
	}
	for i := range want {
		if items[i] != want[i] {
			t.Fatalf("item %d = %#v, want %#v", i, items[i], want[i])
		}
	}
}

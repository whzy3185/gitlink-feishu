package issue

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
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

func TestReadIssueTextArgInline(t *testing.T) {
	ctx := &common.RuntimeContext{Args: map[string]string{"body": "hello"}}
	got, err := readIssueTextArg(ctx, "body", "body-file", true)
	if err != nil {
		t.Fatalf("readIssueTextArg returned error: %v", err)
	}
	if got != "hello" {
		t.Fatalf("readIssueTextArg() = %q, want %q", got, "hello")
	}
}

func TestReadIssueTextArgFromFile(t *testing.T) {
	path := writeTempText(t, "hello from file")
	ctx := &common.RuntimeContext{Args: map[string]string{"body-file": path}}
	got, err := readIssueTextArg(ctx, "body", "body-file", true)
	if err != nil {
		t.Fatalf("readIssueTextArg returned error: %v", err)
	}
	if got != "hello from file" {
		t.Fatalf("readIssueTextArg() = %q, want %q", got, "hello from file")
	}
}

func TestReadIssueTextArgRejectsMixedSources(t *testing.T) {
	path := writeTempText(t, "hello from file")
	ctx := &common.RuntimeContext{Args: map[string]string{"body": "inline", "body-file": path}}
	if _, err := readIssueTextArg(ctx, "body", "body-file", true); err == nil {
		t.Fatal("expected readIssueTextArg to reject mixed inline and file sources")
	}
}

func TestPreviewTextTruncatesLongValue(t *testing.T) {
	input := ""
	for i := 0; i < 150; i++ {
		input += "a"
	}
	got := previewText(input)
	if len([]rune(got)) != 123 {
		t.Fatalf("previewText() length = %d, want %d", len([]rune(got)), 123)
	}
	if got[len(got)-3:] != "..." {
		t.Fatalf("previewText() = %q, want trailing ellipsis", got)
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

func writeTempText(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "body.md")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp text: %v", err)
	}
	return path
}

package issue

import (
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

func writeTempCSV(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "issues.csv")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp csv: %v", err)
	}
	return path
}

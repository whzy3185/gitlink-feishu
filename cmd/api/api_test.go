package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadJSONBodyFromInlineFlag(t *testing.T) {
	cmd := NewAPICmd()
	cmd.Flags().Set("body", `{"title":"hello","count":2}`)

	body, err := readJSONBody(cmd)
	if err != nil {
		t.Fatalf("readJSONBody returned error: %v", err)
	}
	values := body.(map[string]interface{})
	if values["title"] != "hello" {
		t.Fatalf("title = %v, want hello", values["title"])
	}
	if values["count"] != float64(2) {
		t.Fatalf("count = %v, want 2", values["count"])
	}
}

func TestReadJSONBodyFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "body.json")
	if err := os.WriteFile(path, []byte(`{"description":"来自文件"}`), 0o600); err != nil {
		t.Fatalf("write body file: %v", err)
	}

	cmd := NewAPICmd()
	cmd.Flags().Set("body-file", path)

	body, err := readJSONBody(cmd)
	if err != nil {
		t.Fatalf("readJSONBody returned error: %v", err)
	}
	values := body.(map[string]interface{})
	if values["description"] != "来自文件" {
		t.Fatalf("description = %v, want 来自文件", values["description"])
	}
}

func TestReadJSONBodyFromStdin(t *testing.T) {
	cmd := NewAPICmd()
	cmd.Flags().Set("body-stdin", "true")
	cmd.SetIn(strings.NewReader(`{"notes":"from stdin"}`))

	body, err := readJSONBody(cmd)
	if err != nil {
		t.Fatalf("readJSONBody returned error: %v", err)
	}
	values := body.(map[string]interface{})
	if values["notes"] != "from stdin" {
		t.Fatalf("notes = %v, want from stdin", values["notes"])
	}
}

func TestReadJSONBodyRejectsMultipleSources(t *testing.T) {
	cmd := NewAPICmd()
	cmd.Flags().Set("body", `{"title":"hello"}`)
	cmd.Flags().Set("body-stdin", "true")

	if _, err := readJSONBody(cmd); err == nil {
		t.Fatal("expected multiple body sources to return an error")
	}
}

func TestReadJSONBodyRejectsInvalidJSON(t *testing.T) {
	cmd := NewAPICmd()
	cmd.Flags().Set("body", `{"title":`)

	if _, err := readJSONBody(cmd); err == nil {
		t.Fatal("expected invalid JSON to return an error")
	}
}

func TestReadJSONBodyWithoutSource(t *testing.T) {
	cmd := NewAPICmd()

	body, err := readJSONBody(cmd)
	if err != nil {
		t.Fatalf("readJSONBody returned error: %v", err)
	}
	if body != nil {
		t.Fatalf("body = %v, want nil", body)
	}
}

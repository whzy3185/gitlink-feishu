package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrint(t *testing.T) {
	env := SuccessEnvelope(map[string]interface{}{"status": "ok"}, nil)
	if err := Print(env, "json"); err != nil {
		t.Fatalf("Print error: %v", err)
	}
}

func TestPrintDefaultFormat(t *testing.T) {
	env := SuccessEnvelope(map[string]interface{}{"status": "ok"}, nil)
	if err := Print(env, ""); err != nil {
		t.Fatalf("Print default format error: %v", err)
	}
}

func TestPrintToJSON(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope(map[string]interface{}{"status": "ok"}, nil)
	if err := PrintTo(&buf, env, "json"); err != nil {
		t.Fatalf("PrintTo json: %v", err)
	}
	if !strings.Contains(buf.String(), `"ok"`) {
		t.Fatalf("expected ok in JSON output, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), `"status"`) {
		t.Fatalf("expected status in JSON output, got: %s", buf.String())
	}
}

func TestPrintToYAML(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope(map[string]interface{}{"status": "ok"}, nil)
	if err := PrintTo(&buf, env, "yaml"); err != nil {
		t.Fatalf("PrintTo yaml: %v", err)
	}
	if !strings.Contains(buf.String(), "ok") {
		t.Fatalf("expected ok in YAML output, got: %s", buf.String())
	}
}

func TestPrintToTableSlice(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope([]interface{}{
		map[string]interface{}{"id": float64(1), "name": "test"},
		map[string]interface{}{"id": float64(2), "name": "test2"},
	}, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table slice: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "id") || !strings.Contains(out, "name") {
		t.Fatalf("expected table headers, got: %s", out)
	}
	if !strings.Contains(out, "1") || !strings.Contains(out, "test") {
		t.Fatalf("expected table data, got: %s", out)
	}
}

func TestPrintToTableEmptySlice(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope([]interface{}{}, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table empty: %v", err)
	}
	if !strings.Contains(buf.String(), "No results") {
		t.Fatalf("expected 'No results', got: %s", buf.String())
	}
}

func TestPrintToTableSliceNonMap(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope([]interface{}{"string1", "string2"}, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table non-map slice: %v", err)
	}
	// Should fallback to JSON
	if !strings.Contains(buf.String(), "[") {
		t.Fatalf("expected JSON array fallback, got: %s", buf.String())
	}
}

func TestPrintToTableMap(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope(map[string]interface{}{
		"key1": "val1",
		"key2": "val2",
	}, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table map: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "KEY") || !strings.Contains(out, "VALUE") {
		t.Fatalf("expected KEY/VALUE headers, got: %s", out)
	}
}

func TestPrintToTableMapComplex(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope(map[string]interface{}{
		"nested": map[string]interface{}{"a": "b"},
	}, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table complex map: %v", err)
	}
	// Should fallback to JSON because of nested map
	if !strings.Contains(buf.String(), "{") {
		t.Fatalf("expected JSON fallback for complex map, got: %s", buf.String())
	}
}

func TestPrintToTableError(t *testing.T) {
	var buf bytes.Buffer
	env := ErrorEnvelope(500, "server error", "try again")
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Error: server error") {
		t.Fatalf("expected error message, got: %s", out)
	}
	if !strings.Contains(out, "try again") {
		t.Fatalf("expected suggestion, got: %s", out)
	}
}

func TestPrintToTableErrorNoSuggestion(t *testing.T) {
	var buf bytes.Buffer
	env := ErrorEnvelope(500, "server error", "")
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table error no suggestion: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "Suggestion:") {
		t.Fatalf("should not have suggestion line, got: %s", out)
	}
}

func TestPrintToTableNilData(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope(nil, nil)
	if err := PrintTo(&buf, env, "table"); err != nil {
		t.Fatalf("PrintTo table nil data: %v", err)
	}
	if !strings.Contains(buf.String(), "No data") {
		t.Fatalf("expected 'No data', got: %s", buf.String())
	}
}

func TestPrintToDefaultFormat(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope("test", nil)
	if err := PrintTo(&buf, env, ""); err != nil {
		t.Fatalf("PrintTo default format: %v", err)
	}
	// Default should be JSON
	if !strings.Contains(buf.String(), `"ok"`) {
		t.Fatalf("expected JSON output for default format, got: %s", buf.String())
	}
}

func TestPrintToUnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	env := SuccessEnvelope("test", nil)
	if err := PrintTo(&buf, env, "xml"); err != nil {
		t.Fatalf("PrintTo unknown format: %v", err)
	}
	// Unknown format should fallback to JSON
	if !strings.Contains(buf.String(), `"ok"`) {
		t.Fatalf("expected JSON fallback for unknown format, got: %s", buf.String())
	}
}

func TestHasComplexValues(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		want bool
	}{
		{"flat", map[string]interface{}{"a": "1", "b": "2"}, false},
		{"nested map", map[string]interface{}{"a": map[string]interface{}{"x": "y"}}, true},
		{"nested slice", map[string]interface{}{"a": []interface{}{1, 2}}, true},
		{"empty", map[string]interface{}{}, false},
		{"nil", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasComplexValues(tt.m); got != tt.want {
				t.Fatalf("hasComplexValues = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectKeys(t *testing.T) {
	m := map[string]interface{}{
		"title":      "test",
		"id":         float64(1),
		"status":     "open",
		"custom_key": "val",
	}
	keys := collectKeys(m)
	// Priority keys should come first
	if len(keys) != 4 {
		t.Fatalf("expected 4 keys, got %d", len(keys))
	}
	if keys[0] != "id" {
		t.Fatalf("first key should be 'id', got %q", keys[0])
	}
	if keys[1] != "title" {
		t.Fatalf("second key should be 'title', got %q", keys[1])
	}
	if keys[2] != "status" {
		t.Fatalf("third key should be 'status', got %q", keys[2])
	}
}

func TestFormatValue(t *testing.T) {
	tests := []struct {
		name string
		v    interface{}
		want string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"int", 42, "42"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
		{"slice", []interface{}{1, 2, 3}, "[1,2,3]"},
		{"map", map[string]interface{}{"a": "b"}, `{"a":"b"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatValue(tt.v)
			if got != tt.want {
				t.Fatalf("formatValue = %q, want %q", got, tt.want)
			}
		})
	}
}

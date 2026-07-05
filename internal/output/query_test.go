package output

import (
	"bytes"
	"strings"
	"testing"
)

func queryEnvelope() *Envelope {
	return SuccessEnvelope(map[string]interface{}{
		"total_count": 2,
		"commits": []interface{}{
			map[string]interface{}{"sha": "abc", "message": "first"},
			map[string]interface{}{"sha": "def", "message": "second"},
		},
	}, nil)
}

func TestPrintQueryScalarString(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintQuery(&buf, queryEnvelope(), "data.commits.1.sha"); err != nil {
		t.Fatalf("PrintQuery returned error: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "def" {
		t.Fatalf("got %q, want def", got)
	}
}

func TestPrintQueryNumber(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintQuery(&buf, queryEnvelope(), "data.total_count"); err != nil {
		t.Fatalf("PrintQuery returned error: %v", err)
	}
	if got := strings.TrimSpace(buf.String()); got != "2" {
		t.Fatalf("got %q, want 2", got)
	}
}

func TestPrintQueryObjectAsJSON(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintQuery(&buf, queryEnvelope(), "data.commits.0"); err != nil {
		t.Fatalf("PrintQuery returned error: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `"sha": "abc"`) || !strings.Contains(out, `"message": "first"`) {
		t.Fatalf("object output missing fields: %s", out)
	}
}

func TestPrintQueryMissingKeyListsAvailable(t *testing.T) {
	var buf bytes.Buffer
	err := PrintQuery(&buf, queryEnvelope(), "data.nope")
	if err == nil || !strings.Contains(err.Error(), "commits, total_count") {
		t.Fatalf("expected missing-key error listing available keys, got: %v", err)
	}
}

func TestPrintQueryIndexOutOfRange(t *testing.T) {
	var buf bytes.Buffer
	err := PrintQuery(&buf, queryEnvelope(), "data.commits.5")
	if err == nil || !strings.Contains(err.Error(), "out of range") {
		t.Fatalf("expected out-of-range error, got: %v", err)
	}
}

func TestPrintQueryNonIndexSegmentOnArray(t *testing.T) {
	var buf bytes.Buffer
	err := PrintQuery(&buf, queryEnvelope(), "data.commits.sha")
	if err == nil || !strings.Contains(err.Error(), "array index") {
		t.Fatalf("expected array-index error, got: %v", err)
	}
}

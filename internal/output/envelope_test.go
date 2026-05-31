package output

import (
	"encoding/json"
	"testing"
)

func TestSuccessEnvelope(t *testing.T) {
	env := SuccessEnvelope(map[string]interface{}{"key": "value"}, nil)
	if !env.OK {
		t.Fatal("expected OK=true")
	}
	if env.Data == nil {
		t.Fatal("expected non-nil Data")
	}
	if env.Error != nil {
		t.Fatal("expected nil Error")
	}
}

func TestSuccessEnvelopeWithMeta(t *testing.T) {
	meta := &Meta{Page: 1, Limit: 20, TotalCount: 100}
	env := SuccessEnvelope("data", meta)
	if env.Meta != meta {
		t.Fatal("expected Meta to be set")
	}
}

func TestErrorEnvelope(t *testing.T) {
	env := ErrorEnvelope(404, "Not Found", "Check the URL")
	if env.OK {
		t.Fatal("expected OK=false")
	}
	if env.Data != nil {
		t.Fatal("expected nil Data")
	}
	if env.Error == nil {
		t.Fatal("expected non-nil Error")
	}
	if env.Error.Code != 404 {
		t.Fatalf("Code = %v, want 404", env.Error.Code)
	}
	if env.Error.Message != "Not Found" {
		t.Fatalf("Message = %q, want 'Not Found'", env.Error.Message)
	}
	if env.Error.Suggestion != "Check the URL" {
		t.Fatalf("Suggestion = %q, want 'Check the URL'", env.Error.Suggestion)
	}
}

func TestEnvelopeJSON(t *testing.T) {
	env := SuccessEnvelope("hello", nil)
	data, err := env.JSON()
	if err != nil {
		t.Fatalf("JSON() error: %v", err)
	}
	var decoded Envelope
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal JSON output: %v", err)
	}
	if !decoded.OK {
		t.Fatal("expected OK=true in JSON")
	}
}

func TestErrorInfoFields(t *testing.T) {
	info := ErrorInfo{Code: 500, Message: "Internal Error", Suggestion: "Retry later"}
	if info.Message != "Internal Error" {
		t.Fatalf("Message = %q", info.Message)
	}
}

func TestMetaFields(t *testing.T) {
	meta := Meta{Page: 2, Limit: 50, TotalCount: 200, Identity: "user1"}
	if meta.Page != 2 || meta.Limit != 50 || meta.TotalCount != 200 || meta.Identity != "user1" {
		t.Fatal("Meta fields don't match")
	}
}

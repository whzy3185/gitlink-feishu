package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
)

// NewTestServer creates an httptest.Server for shortcut tests.
func NewTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// NewTestContext creates a RuntimeContext wired to the test server.
func NewTestContext(t *testing.T, server *httptest.Server, owner, repo string, args map[string]string) *RuntimeContext {
	t.Helper()
	return &RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  owner,
		Repo:   repo,
		Format: "json",
		Args:   args,
	}
}

// RunShortcut finds and runs a named shortcut.
func RunShortcut(t *testing.T, shortcuts []*Shortcut, name string, ctx *RuntimeContext) error {
	t.Helper()
	for _, s := range shortcuts {
		if s.Name == name {
			return s.Run(ctx)
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

// WriteJSON writes a JSON response to the test response writer.
func WriteJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

// DecodeJSON decodes a JSON request body.
func DecodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	var m map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		return nil
	}
	return m
}

// AssertEqual compares two values.
func AssertEqual(t *testing.T, got, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

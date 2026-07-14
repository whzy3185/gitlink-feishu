package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
)

// NewTestServer creates an httptest.Server with the given handler.
func NewTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// NewTestContext creates a RuntimeContext pointing at the test server.
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

// RunShortcut finds a shortcut by name and runs it with the given context.
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

// DecodeJSON decodes the request body into a map.
func DecodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

// DecodeForm decodes a form-encoded request body into a map with typed values.
func DecodeForm(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read request body: %v", err)
	}
	parsed, err := url.ParseQuery(string(body))
	if err != nil {
		t.Fatalf("failed to parse form body: %v", err)
	}
	result := make(map[string]interface{})
	for k, vs := range parsed {
		if len(vs) == 1 {
			result[k] = vs[0]
		} else {
			result[k] = vs
		}
	}
	return result
}

// WriteJSON writes a JSON response.
func WriteJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

// AssertEqual compares two values.
func AssertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

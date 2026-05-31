package common

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

func TestRuntimeContextRepoPath(t *testing.T) {
	ctx := &RuntimeContext{Owner: "owner", Repo: "repo"}
	if got := ctx.RepoPath(); got != "/owner/repo" {
		t.Fatalf("RepoPath = %q, want /owner/repo", got)
	}
}

func TestRuntimeContextArg(t *testing.T) {
	ctx := &RuntimeContext{
		Args: map[string]string{"key1": "val1"},
	}

	tests := []struct {
		name string
		want string
	}{
		{"key1", "val1"},
		{"key2", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ctx.Arg(tt.name); got != tt.want {
				t.Fatalf("Arg(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}

func TestRuntimeContextRequireArg(t *testing.T) {
	ctx := &RuntimeContext{
		Args: map[string]string{"required": "present"},
	}

	v, err := ctx.RequireArg("required")
	if err != nil {
		t.Fatalf("RequireArg error: %v", err)
	}
	if v != "present" {
		t.Fatalf("RequireArg = %q, want present", v)
	}

	_, err = ctx.RequireArg("missing")
	if err == nil {
		t.Fatal("expected error for missing required arg")
	}
}

func TestRuntimeContextCallAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":"test"}`))
	}))
	defer server.Close()

	ctx := &RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
	}

	env, err := ctx.CallAPI("GET", "/test", nil)
	if err != nil {
		t.Fatalf("CallAPI error: %v", err)
	}
	if !env.OK {
		t.Fatal("expected OK=true")
	}
}

func TestRuntimeContextCallAPIWithQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "open" {
			t.Fatalf("expected state=open, got %s", r.URL.Query().Get("state"))
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":"test"}`))
	}))
	defer server.Close()

	ctx := &RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
	}

	q := url.Values{}
	q.Set("state", "open")
	_, err := ctx.CallAPIWithQuery("GET", "/test", q)
	if err != nil {
		t.Fatalf("CallAPIWithQuery error: %v", err)
	}
}

func TestRuntimeContextOutput(t *testing.T) {
	ctx := &RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "json",
	}
	// Output should succeed (data goes to stdout)
	env := &output.Envelope{OK: true, Data: map[string]interface{}{"key": "val"}}
	err := ctx.Output(env)
	if err != nil {
		t.Fatalf("Output error: %v", err)
	}
}

func TestRuntimeContextOutputData(t *testing.T) {
	ctx := &RuntimeContext{
		Client: &client.Client{HTTP: http.DefaultClient, BaseURL: "http://localhost"},
		Format: "json",
	}
	err := ctx.OutputData(map[string]interface{}{"key": "val"})
	if err != nil {
		t.Fatalf("OutputData error: %v", err)
	}
}

func TestRuntimeContextResolveOwnerRepo(t *testing.T) {
	// When Owner and Repo are already set, ResolveOwnerRepo should succeed
	ctx := &RuntimeContext{Owner: "explicitOwner", Repo: "explicitRepo"}
	if err := ctx.ResolveOwnerRepo(); err != nil {
		t.Fatalf("ResolveOwnerRepo error: %v", err)
	}
	if ctx.Owner != "explicitOwner" || ctx.Repo != "explicitRepo" {
		t.Fatalf("ResolveOwnerRepo changed values")
	}
}

func TestShortcutStruct(t *testing.T) {
	s := Shortcut{
		Name:        "test",
		Description: "test shortcut",
		Flags:       []Flag{{Name: "verbose", Usage: "verbose output"}},
		Run:         func(ctx *RuntimeContext) error { return nil },
	}
	if s.Name != "test" {
		t.Fatalf("Name = %q", s.Name)
	}
	if len(s.Flags) != 1 {
		t.Fatalf("expected 1 flag, got %d", len(s.Flags))
	}
	if s.Run == nil {
		t.Fatal("expected non-nil Run")
	}
}

func TestFlagStruct(t *testing.T) {
	f := Flag{
		Name:     "output",
		Short:    "o",
		Usage:    "Output format",
		Required: true,
		Default:  "json",
		Bool:     false,
	}
	if f.Name != "output" {
		t.Fatalf("Name = %q", f.Name)
	}
	if f.Short != "o" {
		t.Fatalf("Short = %q", f.Short)
	}
	if !f.Required {
		t.Fatal("expected Required=true")
	}
}

func TestRuntimeContextPaginateAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":1},{"id":2}]}`))
	}))
	defer server.Close()

	ctx := &RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
	}
	items, err := ctx.PaginateAll("/items", nil)
	if err != nil {
		t.Fatalf("PaginateAll error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("PaginateAll returned %d items, want 2", len(items))
	}
}

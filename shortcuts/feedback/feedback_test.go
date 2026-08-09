package feedback

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestFeedbackCreateWithExplicitUser(t *testing.T) {
	var payload map[string]interface{}
	server := newFeedbackServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertFeedbackRequest(t, r, "POST", "/v1/Mengz/feedbacks.json")
		payload = decodeFeedbackJSON(t, r)
		writeFeedbackJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runFeedbackShortcut(t, server, "create", map[string]string{
		"user":     "Mengz",
		"content":  "The CLI should support feedback.",
		"category": "feature",
		"contact":  "mengz@example.com",
		"repo-ref": "Gitlink/gitlink-cli",
	})
	if err != nil {
		t.Fatalf("feedback create failed: %v", err)
	}
	content, _ := payload["content"].(string)
	for _, want := range []string{
		"Category: feature",
		"Repository: Gitlink/gitlink-cli",
		"Contact: mengz@example.com",
		"The CLI should support feedback.",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("content missing %q: %q", want, content)
		}
	}
}

func TestFeedbackCreateDefaultsToCurrentUser(t *testing.T) {
	var paths []string
	server := newFeedbackServer(t, func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		switch r.URL.Path {
		case "/users/me.json":
			assertFeedbackRequest(t, r, "GET", "/users/me.json")
			writeFeedbackJSON(t, w, map[string]interface{}{"login": "Mengz"})
		case "/v1/Mengz/feedbacks.json":
			assertFeedbackRequest(t, r, "POST", "/v1/Mengz/feedbacks.json")
			writeFeedbackJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runFeedbackShortcut(t, server, "create", map[string]string{"content": "Feedback body"})
	if err != nil {
		t.Fatalf("feedback create failed: %v", err)
	}
	want := []string{"/users/me.json", "/v1/Mengz/feedbacks.json"}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
}

func TestFeedbackCreateReadsFileAndStdin(t *testing.T) {
	oldInput := feedbackInput
	feedbackInput = strings.NewReader("stdin details\n")
	defer func() { feedbackInput = oldInput }()

	var payload map[string]interface{}
	server := newFeedbackServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertFeedbackRequest(t, r, "POST", "/v1/Mengz/feedbacks.json")
		payload = decodeFeedbackJSON(t, r)
		writeFeedbackJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	path := filepath.Join(t.TempDir(), "feedback.txt")
	if err := os.WriteFile(path, []byte("file details\n"), 0o600); err != nil {
		t.Fatalf("write feedback file: %v", err)
	}
	err := runFeedbackShortcut(t, server, "create", map[string]string{
		"user":  "Mengz",
		"from":  path,
		"stdin": "true",
	})
	if err != nil {
		t.Fatalf("feedback create failed: %v", err)
	}
	content, _ := payload["content"].(string)
	if !strings.Contains(content, "file details") || !strings.Contains(content, "stdin details") {
		t.Fatalf("content = %q, want file and stdin details", content)
	}
}

func TestFeedbackCreateDryRunDoesNotCallSubmitAPI(t *testing.T) {
	server := newFeedbackServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("dry-run should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runFeedbackShortcut(t, server, "create", map[string]string{
		"user":    "Mengz",
		"content": "Preview this feedback",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("feedback dry-run failed: %v", err)
	}
}

func TestFeedbackCreateRejectsEmptyContentBeforeAPI(t *testing.T) {
	server := newFeedbackServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("empty content should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runFeedbackShortcut(t, server, "create", map[string]string{"user": "Mengz"})
	if err == nil {
		t.Fatal("expected empty feedback to return an error")
	}
}

func TestBuildFeedbackContentAddsRepositoryFromContext(t *testing.T) {
	ctx := &common.RuntimeContext{
		Owner: "Gitlink",
		Repo:  "gitlink-cli",
		Args:  map[string]string{"content": "Feedback body"},
	}
	content, err := buildFeedbackContent(ctx)
	if err != nil {
		t.Fatalf("buildFeedbackContent failed: %v", err)
	}
	if !strings.Contains(content, "Repository: Gitlink/gitlink-cli") {
		t.Fatalf("content = %q, want repository metadata", content)
	}
}

func runFeedbackShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findFeedbackShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findFeedbackShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newFeedbackServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertFeedbackRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeFeedbackJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	return payload
}

func writeFeedbackJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

package pr

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPRReviewCommentsIsReadOnlyRegistryBoundary(t *testing.T) {
	names := map[string]bool{}
	for _, shortcut := range Shortcuts() {
		names[shortcut.Name] = true
	}

	if !names["review-comments"] {
		t.Fatal("Shortcuts missing read-only review-comments")
	}
	for _, name := range []string{"review-comment", "review-comment-update", "review-comment-delete"} {
		if names[name] {
			t.Errorf("write shortcut %q must stay unregistered before API verification", name)
		}
	}
}

func TestPRReviewCommentsRequestContract(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/v1/owner/repo/pulls/42/journals.json" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		query := r.URL.Query()
		expected := map[string]string{
			"keyword":        "panic",
			"review_id":      "7",
			"need_respond":   "true",
			"state":          "opened",
			"parent_id":      "11",
			"path":           "目录/main.go",
			"is_full":        "true",
			"sort_by":        "updated_on",
			"sort_direction": "desc",
		}
		for key, want := range expected {
			if got := query.Get(key); got != want {
				t.Errorf("query %s = %q, want %q", key, got, want)
			}
		}
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "journals": []interface{}{}})
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review-comments", map[string]string{
		"id":             "42",
		"keyword":        "panic",
		"review-id":      "7",
		"need-respond":   "true",
		"state":          "opened",
		"parent-id":      "11",
		"path":           "目录/main.go",
		"is-full":        "true",
		"sort-by":        "updated_on",
		"sort-direction": "desc",
	})
	if err != nil {
		t.Fatalf("review-comments failed: %v", err)
	}
}

func TestPRReviewCommentsRejectsInvalidBooleanBeforeRequest(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}))
	defer server.Close()

	err := runPRShortcut(t, server, "review-comments", map[string]string{
		"id":           "42",
		"need-respond": "sometimes",
	})
	if err == nil {
		t.Fatal("expected invalid boolean error")
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0", requests)
	}
}

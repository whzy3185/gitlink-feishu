package label

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestLabelList(t *testing.T) {
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/issue_tags.json")
		if got := r.URL.Query().Get("keyword"); got != "bug" {
			t.Fatalf("got keyword %q, want %q", got, "bug")
		}
		if got := r.URL.Query().Get("order_by"); got != "issues_count" {
			t.Fatalf("got order_by %q, want %q", got, "issues_count")
		}
		writeJSON(t, w, map[string]interface{}{"total_count": 0, "issue_tags": []interface{}{}})
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "list", map[string]string{
		"keyword": "bug",
		"sort-by": "issues_count",
	})
	if err != nil {
		t.Fatalf("list shortcut failed: %v", err)
	}
}

func TestLabelCreatePayload(t *testing.T) {
	var payload map[string]interface{}
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/v1/owner/repo/issue_tags.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "create", map[string]string{
		"name":        "bug",
		"description": "Something is broken",
		"color":       "#FF0000",
	})
	if err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}

	assertEqual(t, payload["name"], "bug")
	assertEqual(t, payload["description"], "Something is broken")
	assertEqual(t, payload["color"], "#FF0000")
}

func TestLabelCreateUsesDefaultColor(t *testing.T) {
	var payload map[string]interface{}
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/v1/owner/repo/issue_tags.json")
		payload = decodeJSON(t, r)
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runLabelShortcut(t, server, "create", map[string]string{"name": "enhancement"}); err != nil {
		t.Fatalf("create shortcut failed: %v", err)
	}
	assertEqual(t, payload["color"], defaultLabelColor)
}

func TestLabelCreateRejectsInvalidColor(t *testing.T) {
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid color should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "create", map[string]string{
		"name":  "bug",
		"color": "red",
	})
	if err == nil {
		t.Fatal("expected invalid color to return an error")
	}
}

func TestLabelUpdatePreservesCurrentFields(t *testing.T) {
	var payload map[string]interface{}
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"issue_tags": []interface{}{
					map[string]interface{}{
						"id":          float64(7),
						"name":        "bug",
						"description": "old description",
						"color":       "#FF0000",
					},
				},
			})
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issue_tags/7.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "update", map[string]string{
		"id":    "7",
		"color": "#00FF00",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	// name and description preserved from current; only color changed.
	assertEqual(t, payload["name"], "bug")
	assertEqual(t, payload["description"], "old description")
	assertEqual(t, payload["color"], "#00FF00")
}

func TestLabelUpdatePreservesFieldsWhenLabelOnSecondPage(t *testing.T) {
	var payload map[string]interface{}
	var pagesFetched []string
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			page := r.URL.Query().Get("page")
			pagesFetched = append(pagesFetched, page)
			switch page {
			case "1":
				writeJSON(t, w, map[string]interface{}{
					"total_count": labelListPageSize + 1,
					"issue_tags":  fillerLabels(labelListPageSize),
				})
			case "2":
				writeJSON(t, w, map[string]interface{}{
					"total_count": labelListPageSize + 1,
					"issue_tags": []interface{}{
						map[string]interface{}{
							"id":          float64(7),
							"name":        "bug",
							"description": "old description",
							"color":       "#FF0000",
						},
					},
				})
			default:
				t.Fatalf("unexpected page %q", page)
			}
		case r.Method == "PATCH" && r.URL.Path == "/v1/owner/repo/issue_tags/7.json":
			payload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "update", map[string]string{
		"id":   "7",
		"name": "renamed",
	})
	if err != nil {
		t.Fatalf("update shortcut failed: %v", err)
	}

	if len(pagesFetched) != 2 || pagesFetched[0] != "1" || pagesFetched[1] != "2" {
		t.Fatalf("expected pages [1 2] to be fetched, got %v", pagesFetched)
	}
	// The label lives on page 2; only --name was passed, so the description and
	// color the server already holds must survive the PATCH untouched.
	assertEqual(t, payload["name"], "renamed")
	assertEqual(t, payload["description"], "old description")
	assertEqual(t, payload["color"], "#FF0000")
}

func TestLabelUpdateErrorsWhenLabelNotFound(t *testing.T) {
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"issue_tags": []interface{}{
					map[string]interface{}{
						"id":          float64(3),
						"name":        "docs",
						"description": "documentation",
						"color":       "#00FF00",
					},
				},
			})
		default:
			t.Fatalf("missing label must not PATCH, got: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "update", map[string]string{
		"id":    "7",
		"color": "#123456",
	})
	if err == nil {
		t.Fatal("expected update of a missing label id to return an error")
	}
}

func TestLabelUpdateRequiresAtLeastOneField(t *testing.T) {
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("update with no fields should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	err := runLabelShortcut(t, server, "update", map[string]string{"id": "7"})
	if err == nil {
		t.Fatal("expected update with no fields to return an error")
	}
}

func TestLabelDelete(t *testing.T) {
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/v1/owner/repo/issue_tags/7.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runLabelShortcut(t, server, "delete", map[string]string{"id": "7"}); err != nil {
		t.Fatalf("delete shortcut failed: %v", err)
	}
}

func TestValidateColor(t *testing.T) {
	valid := []string{"#1E90FF", "#abc", "#ABCDEF", "#000"}
	for _, c := range valid {
		if err := validateColor(c); err != nil {
			t.Fatalf("expected %q to be valid, got %v", c, err)
		}
	}
	invalid := []string{"red", "1E90FF", "#12", "#GGGGGG", "#1234", ""}
	for _, c := range invalid {
		if err := validateColor(c); err == nil {
			t.Fatalf("expected %q to be invalid", c)
		}
	}
}

func TestLabelIDString(t *testing.T) {
	assertEqual(t, labelIDString(float64(7)), "7")
	assertEqual(t, labelIDString("9"), "9")
	assertEqual(t, labelIDString(json.Number("11")), "11")
	assertEqual(t, labelIDString(nil), "")
}

func runLabelShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findLabelShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{
			HTTP:    server.Client(),
			BaseURL: server.URL,
		},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}
	if ctx.Args == nil {
		ctx.Args = map[string]string{}
	}
	return shortcut.Run(ctx)
}

func findLabelShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newLabelTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

// fillerLabels builds n distinct labels whose ids never collide with the target
// ids used in the update tests, so a full page forces fetchLabel onto the next.
func fillerLabels(n int) []interface{} {
	labels := make([]interface{}, 0, n)
	for i := 0; i < n; i++ {
		labels = append(labels, map[string]interface{}{
			"id":          float64(1000 + i),
			"name":        fmt.Sprintf("filler-%d", i),
			"description": "filler",
			"color":       "#123456",
		})
	}
	return labels
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("got request %s %s, want %s %s", r.Method, r.URL.Path, method, path)
	}
}

func decodeJSON(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("failed to decode request body: %v", err)
	}
	return payload
}

func writeJSON(t *testing.T, w http.ResponseWriter, payload interface{}) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("failed to write response: %v", err)
	}
}

func assertEqual(t *testing.T, got interface{}, want interface{}) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

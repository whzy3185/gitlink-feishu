package label

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
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

func TestLabelCloneSkipsExistingCreatesNew(t *testing.T) {
	var posted []map[string]interface{}
	patched := false
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/src-owner/src-repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 2,
				"issue_tags": []interface{}{
					map[string]interface{}{"id": float64(1), "name": "bug", "description": "b", "color": "#FF0000"},
					map[string]interface{}{"id": float64(2), "name": "feature", "description": "f", "color": "#00FF00"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"issue_tags": []interface{}{
					map[string]interface{}{"id": float64(9), "name": "bug", "description": "existing", "color": "#123456"},
				},
			})
		case r.Method == "POST" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			posted = append(posted, decodeJSON(t, r))
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		case r.Method == "PATCH":
			patched = true
			t.Fatalf("unexpected PATCH without --force: %s", r.URL.Path)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runLabelShortcut(t, server, "clone", map[string]string{"source": "src-owner/src-repo"}); err != nil {
		t.Fatalf("clone shortcut failed: %v", err)
	}
	if patched {
		t.Fatal("expected no PATCH without --force")
	}
	if len(posted) != 1 {
		t.Fatalf("expected 1 created label, got %d", len(posted))
	}
	// The colliding "bug" is skipped by name; only "feature" is created, with
	// the source's own color carried over.
	assertEqual(t, posted[0]["name"], "feature")
	assertEqual(t, posted[0]["color"], "#00FF00")
}

func TestLabelCloneForceUpdatesExisting(t *testing.T) {
	var patchPath string
	var patchPayload map[string]interface{}
	posted := false
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" && r.URL.Path == "/v1/src-owner/src-repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"issue_tags": []interface{}{
					map[string]interface{}{"id": float64(1), "name": "bug", "description": "from source", "color": "#FF0000"},
				},
			})
		case r.Method == "GET" && r.URL.Path == "/v1/owner/repo/issue_tags.json":
			writeJSON(t, w, map[string]interface{}{
				"total_count": 1,
				"issue_tags": []interface{}{
					map[string]interface{}{"id": float64(9), "name": "bug", "description": "old", "color": "#000000"},
				},
			})
		case r.Method == "PATCH":
			patchPath = r.URL.Path
			patchPayload = decodeJSON(t, r)
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		case r.Method == "POST":
			posted = true
			t.Fatalf("unexpected POST for an existing label under --force")
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runLabelShortcut(t, server, "clone", map[string]string{"source": "src-owner/src-repo", "force": "true"}); err != nil {
		t.Fatalf("clone shortcut failed: %v", err)
	}
	if posted {
		t.Fatal("expected no POST for an existing label under --force")
	}
	// --force PATCHes the existing label id in place so issue associations
	// survive, and overwrites its fields from the source.
	assertEqual(t, patchPath, "/v1/owner/repo/issue_tags/9.json")
	assertEqual(t, patchPayload["name"], "bug")
	assertEqual(t, patchPayload["description"], "from source")
	assertEqual(t, patchPayload["color"], "#FF0000")
}

func TestFetchLabelsForRepoPaginates(t *testing.T) {
	pagesSeen := map[string]bool{}
	server := newLabelTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/v1/owner/repo/issue_tags.json")
		page := r.URL.Query().Get("page")
		pagesSeen[page] = true
		if got := r.URL.Query().Get("limit"); got != strconv.Itoa(labelPageSize) {
			t.Fatalf("got limit %q, want %d", got, labelPageSize)
		}
		var tags []interface{}
		switch page {
		case "1":
			tags = make([]interface{}, labelPageSize)
			for i := range tags {
				tags[i] = map[string]interface{}{"id": float64(i + 1), "name": fmt.Sprintf("l%d", i+1)}
			}
		case "2":
			tags = []interface{}{
				map[string]interface{}{"id": float64(101), "name": "l101"},
				map[string]interface{}{"id": float64(102), "name": "l102"},
				map[string]interface{}{"id": float64(103), "name": "l103"},
			}
		default:
			t.Fatalf("unexpected page %q", page)
		}
		writeJSON(t, w, map[string]interface{}{"total_count": labelPageSize + 3, "issue_tags": tags})
	})
	defer server.Close()

	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   map[string]string{},
	}
	labels, err := fetchLabelsForRepo(ctx, "owner", "repo")
	if err != nil {
		t.Fatalf("fetchLabelsForRepo error: %v", err)
	}
	if len(labels) != labelPageSize+3 {
		t.Fatalf("got %d labels, want %d", len(labels), labelPageSize+3)
	}
	if !pagesSeen["1"] || !pagesSeen["2"] {
		t.Fatalf("expected pages 1 and 2 to be walked, saw %v", pagesSeen)
	}
}

func TestSplitOwnerRepo(t *testing.T) {
	cases := []struct {
		in        string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"/owner/repo/", "owner", "repo", false},
		{" owner/repo ", "owner", "repo", false},
		{"owner/repo/sub", "owner", "repo", false},
		{"owner", "", "", true},
		{"", "", "", true},
		{"/", "", "", true},
	}
	for _, tc := range cases {
		owner, repo, err := splitOwnerRepo(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("splitOwnerRepo(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("splitOwnerRepo(%q) unexpected error: %v", tc.in, err)
		}
		assertEqual(t, owner, tc.wantOwner)
		assertEqual(t, repo, tc.wantRepo)
	}
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

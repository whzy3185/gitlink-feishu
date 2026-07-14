package org

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func runShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	_, err := runShortcutOutput(t, server, name, args)
	return err
}

func runShortcutOutput(t *testing.T, server *httptest.Server, name string, args map[string]string) (string, error) {
	t.Helper()
	shortcut := findShortcut(t, name)
	ctx := &common.RuntimeContext{
		Client: &client.Client{HTTP: server.Client(), BaseURL: server.URL},
		Owner:  "owner",
		Repo:   "repo",
		Format: "json",
		Args:   args,
	}

	oldStdout := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe stdout: %v", err)
	}

	os.Stdout = writer
	runErr := shortcut.Run(ctx)
	writer.Close()
	os.Stdout = oldStdout

	output, readErr := io.ReadAll(reader)
	reader.Close()
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}

	return string(output), runErr
}

func findShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, s := range Shortcuts() {
		if s.Name == name {
			return s
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		panic(err)
	}
}

func parseEnvelope(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var env map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &env); err != nil {
		t.Fatalf("parse envelope: %v\nraw=%s", err, raw)
	}
	return env
}

func decodeJSONBody(t *testing.T, r *http.Request) map[string]interface{} {
	t.Helper()
	defer r.Body.Close()
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return payload
}

func fixtureOrganizationUser(id int, login, name, mail string, teamNames ...string) map[string]interface{} {
	teams := make([]interface{}, 0, len(teamNames))
	for _, teamName := range teamNames {
		teams = append(teams, teamName)
	}
	return map[string]interface{}{
		"id": float64(id),
		"user": map[string]interface{}{
			"login":    login,
			"name":     name,
			"identity": login + "-identity",
			"mail":     mail,
		},
		"team_names": teams,
		"created_at": "2026-06-10T00:00:00Z",
	}
}

func fixtureTeam(id int, name, nickname, description, authorize string, units []string, users ...map[string]interface{}) map[string]interface{} {
	unitValues := make([]interface{}, 0, len(units))
	for _, unit := range units {
		unitValues = append(unitValues, unit)
	}
	userValues := make([]interface{}, 0, len(users))
	for _, user := range users {
		userValues = append(userValues, user)
	}
	return map[string]interface{}{
		"id":                     float64(id),
		"name":                   name,
		"nickname":               nickname,
		"description":            description,
		"authorize":              authorize,
		"includes_all_project":   false,
		"can_create_org_project": true,
		"num_projects":           float64(3),
		"num_users":              float64(len(users)),
		"units":                  unitValues,
		"users":                  userValues,
	}
}

func fixtureTeamUser(userID int, login, name, mail string) map[string]interface{} {
	return map[string]interface{}{
		"user_id":  float64(userID),
		"login":    login,
		"name":     name,
		"identity": login + "-identity",
		"mail":     mail,
	}
}

func TestOrgList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "2" || r.URL.Query().Get("limit") != "50" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		writeJSON(w, []interface{}{
			map[string]interface{}{"login": "org1"},
			map[string]interface{}{"login": "org2"},
		})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"page": "2", "limit": "50"}); err != nil {
		t.Fatalf("list failed: %v", err)
	}
}

func TestOrgInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"login": "myorg", "name": "My Org"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "info", map[string]string{"id": "myorg"}); err != nil {
		t.Fatalf("info failed: %v", err)
	}
}

func TestOrgMembersUsesRequestedPageWithoutFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg/organization_users.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("page") != "3" || r.URL.Query().Get("limit") != "2" {
			t.Fatalf("unexpected query: %s", r.URL.RawQuery)
		}
		writeJSON(w, map[string]interface{}{
			"organization_users": []interface{}{
				fixtureOrganizationUser(12, "alice", "Alice", "alice@example.com", "Platform"),
			},
			"total_count": float64(9),
		})
	}))
	defer server.Close()

	output, err := runShortcutOutput(t, server, "members", map[string]string{
		"id":    "myorg",
		"page":  "3",
		"limit": "2",
	})
	if err != nil {
		t.Fatalf("members failed: %v", err)
	}

	env := parseEnvelope(t, output)
	data := env["data"].(map[string]interface{})
	if data["fetched_all"] != false {
		t.Fatalf("fetched_all = %v, want false", data["fetched_all"])
	}
	if data["total_count"].(float64) != 9 {
		t.Fatalf("total_count = %v, want 9", data["total_count"])
	}
	users := data["organization_users"].([]interface{})
	if len(users) != 1 {
		t.Fatalf("organization_users len = %d, want 1", len(users))
	}
	user := users[0].(map[string]interface{})
	if user["login"] != "alice" {
		t.Fatalf("login = %v, want alice", user["login"])
	}
}

func TestOrgMembersFiltersFetchAllPages(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if r.URL.Path != "/organizations/myorg/organization_users.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		switch r.URL.Query().Get("page") {
		case "1":
			writeJSON(w, map[string]interface{}{
				"organization_users": []interface{}{
					fixtureOrganizationUser(11, "zoe", "Zoe", "zoe@example.com", "Ops"),
				},
				"total_count": float64(2),
			})
		case "2":
			writeJSON(w, map[string]interface{}{
				"organization_users": []interface{}{
					fixtureOrganizationUser(12, "alice", "Alice", "alice@example.com", "Platform"),
				},
				"total_count": float64(2),
			})
		default:
			t.Fatalf("unexpected page: %s", r.URL.Query().Get("page"))
		}
	}))
	defer server.Close()

	output, err := runShortcutOutput(t, server, "members", map[string]string{
		"id":      "myorg",
		"page":    "1",
		"limit":   "1",
		"keyword": "alice",
	})
	if err != nil {
		t.Fatalf("members with filters failed: %v", err)
	}
	if callCount != 2 {
		t.Fatalf("callCount = %d, want 2", callCount)
	}

	env := parseEnvelope(t, output)
	data := env["data"].(map[string]interface{})
	if data["fetched_all"] != true {
		t.Fatalf("fetched_all = %v, want true", data["fetched_all"])
	}
	if data["matched_count"].(float64) != 1 {
		t.Fatalf("matched_count = %v, want 1", data["matched_count"])
	}
	filters := data["filters"].(map[string]interface{})
	if filters["keyword"] != "alice" {
		t.Fatalf("keyword filter = %v, want alice", filters["keyword"])
	}
	counts := data["counts_by_team"].(map[string]interface{})
	if counts["Platform"].(float64) != 1 {
		t.Fatalf("counts_by_team[Platform] = %v, want 1", counts["Platform"])
	}
	users := data["organization_users"].([]interface{})
	user := users[0].(map[string]interface{})
	if user["login"] != "alice" {
		t.Fatalf("login = %v, want alice", user["login"])
	}
}

func TestOrgTeamsFiltersAndIncludeUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg/teams.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"total_count": float64(2),
			"teams": []interface{}{
				fixtureTeam(1, "platform", "Platform", "Platform engineering team", "write", []string{"R&D"},
					fixtureTeamUser(102, "zoe", "Zoe", "zoe@example.com"),
					fixtureTeamUser(101, "alice", "Alice", "alice@example.com"),
				),
				fixtureTeam(2, "ops", "Ops", "Operations team", "read", []string{"Ops"},
					fixtureTeamUser(103, "bob", "Bob", "bob@example.com"),
				),
			},
		})
	}))
	defer server.Close()

	output, err := runShortcutOutput(t, server, "teams", map[string]string{
		"id":            "myorg",
		"authorize":     "write",
		"keyword":       "platform",
		"include-users": "true",
	})
	if err != nil {
		t.Fatalf("teams failed: %v", err)
	}

	env := parseEnvelope(t, output)
	data := env["data"].(map[string]interface{})
	if data["matched_count"].(float64) != 1 {
		t.Fatalf("matched_count = %v, want 1", data["matched_count"])
	}
	if data["include_users"] != true {
		t.Fatalf("include_users = %v, want true", data["include_users"])
	}
	counts := data["counts_by_authorize"].(map[string]interface{})
	if counts["write"].(float64) != 1 {
		t.Fatalf("counts_by_authorize[write] = %v, want 1", counts["write"])
	}
	teams := data["teams"].([]interface{})
	if len(teams) != 1 {
		t.Fatalf("teams len = %d, want 1", len(teams))
	}
	team := teams[0].(map[string]interface{})
	memberLogins := team["member_logins"].([]interface{})
	if memberLogins[0] != "alice" || memberLogins[1] != "zoe" {
		t.Fatalf("member_logins = %v, want sorted [alice zoe]", memberLogins)
	}
	users := team["users"].([]interface{})
	if users[0].(map[string]interface{})["login"] != "alice" {
		t.Fatalf("users not sorted by login: %v", users)
	}
}

func TestOrgCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/organizations.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload := decodeJSONBody(t, r)
		if payload["name"] != "neworg" {
			t.Fatalf("name = %v, want neworg", payload["name"])
		}
		if payload["description"] != "A new org" {
			t.Fatalf("description = %v, want A new org", payload["description"])
		}
		writeJSON(w, map[string]interface{}{"login": "neworg"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "create", map[string]string{"name": "neworg", "description": "A new org"}); err != nil {
		t.Fatalf("create failed: %v", err)
	}
}

func TestOrgTeamCreateDryRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected for dry-run")
	}))
	defer server.Close()

	output, err := runShortcutOutput(t, server, "team-create", map[string]string{
		"id":          "myorg",
		"name":        "platform",
		"nickname":    "Platform",
		"description": "Core team",
		"dry-run":     "true",
	})
	if err != nil {
		t.Fatalf("team-create dry-run failed: %v", err)
	}

	env := parseEnvelope(t, output)
	data := env["data"].(map[string]interface{})
	if data["method"] != "POST" {
		t.Fatalf("method = %v, want POST", data["method"])
	}
	if data["path"] != "/organizations/myorg/teams" {
		t.Fatalf("path = %v, want /organizations/myorg/teams", data["path"])
	}
	payload := data["payload"].(map[string]interface{})
	if payload["name"] != "platform" || payload["nickname"] != "Platform" || payload["description"] != "Core team" {
		t.Fatalf("unexpected payload: %v", payload)
	}
}

func TestOrgTeamCreatePostsPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg/teams.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		payload := decodeJSONBody(t, r)
		if payload["name"] != "platform" {
			t.Fatalf("name = %v, want platform", payload["name"])
		}
		if payload["nickname"] != "Platform" {
			t.Fatalf("nickname = %v, want Platform", payload["nickname"])
		}
		if payload["description"] != "Core team" {
			t.Fatalf("description = %v, want Core team", payload["description"])
		}
		writeJSON(w, map[string]interface{}{"id": float64(7), "name": "platform"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "team-create", map[string]string{
		"id":          "myorg",
		"name":        "platform",
		"nickname":    "Platform",
		"description": "Core team",
	}); err != nil {
		t.Fatalf("team-create failed: %v", err)
	}
}

func TestOrgMemberRemoveDryRunByLogin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET during login resolution, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg/organization_users.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{
			"organization_users": []interface{}{
				fixtureOrganizationUser(9, "alice", "Alice", "alice@example.com", "Platform"),
			},
			"total_count": float64(1),
		})
	}))
	defer server.Close()

	output, err := runShortcutOutput(t, server, "member-remove", map[string]string{
		"id":      "myorg",
		"login":   "alice",
		"dry-run": "true",
	})
	if err != nil {
		t.Fatalf("member-remove dry-run failed: %v", err)
	}

	env := parseEnvelope(t, output)
	data := env["data"].(map[string]interface{})
	if data["method"] != "DELETE" {
		t.Fatalf("method = %v, want DELETE", data["method"])
	}
	if data["path"] != "/organizations/myorg/organization_users/9" {
		t.Fatalf("path = %v, want /organizations/myorg/organization_users/9", data["path"])
	}
	member := data["member"].(map[string]interface{})
	if member["login"] != "alice" {
		t.Fatalf("member.login = %v, want alice", member["login"])
	}
}

func TestOrgMemberRemoveByUserID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/organizations/myorg/organization_users/9.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(w, map[string]interface{}{"status": 0, "message": "removed"})
	}))
	defer server.Close()

	if err := runShortcut(t, server, "member-remove", map[string]string{
		"id":      "myorg",
		"user-id": "9",
	}); err != nil {
		t.Fatalf("member-remove failed: %v", err)
	}
}

func TestOrgMembersInvalidPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "members", map[string]string{
		"id":    "myorg",
		"page":  "0",
		"limit": "20",
	})
	if err == nil || !strings.Contains(err.Error(), "--page must be a positive integer") {
		t.Fatalf("expected positive integer error, got %v", err)
	}
}

func TestOrgMemberRemoveRequiresTarget(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("no API call expected")
	}))
	defer server.Close()

	err := runShortcut(t, server, "member-remove", map[string]string{"id": "myorg"})
	if err == nil || !strings.Contains(err.Error(), "one of --user-id or --login is required") {
		t.Fatalf("expected missing target error, got %v", err)
	}
}

func TestOrgListHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "list", map[string]string{"page": "1", "limit": "20"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

func TestOrgTeamsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	if err := runShortcut(t, server, "teams", map[string]string{"id": "myorg"}); err == nil {
		t.Fatal("expected error for HTTP 500")
	}
}

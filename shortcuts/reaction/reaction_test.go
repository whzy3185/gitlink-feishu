package reaction

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func TestReactionWatchersBuildsTimeRangeQuery(t *testing.T) {
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/watchers.json")
		assertEqual(t, r.URL.Query().Get("start_at"), "1700000000")
		assertEqual(t, r.URL.Query().Get("end_at"), "1700003600")
		writeJSON(t, w, map[string]interface{}{"count": 0, "users": []interface{}{}})
	})
	defer server.Close()

	err := runReactionShortcut(t, server, "watchers", map[string]string{
		"start-at": "1700000000",
		"end-at":   "1700003600",
	})
	if err != nil {
		t.Fatalf("watchers shortcut failed: %v", err)
	}
}

func TestReactionStargazers(t *testing.T) {
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "GET", "/owner/repo/stargazers.json")
		writeJSON(t, w, map[string]interface{}{"count": 1, "users": []interface{}{}})
	})
	defer server.Close()

	if err := runReactionShortcut(t, server, "stargazers", nil); err != nil {
		t.Fatalf("stargazers shortcut failed: %v", err)
	}
}

func TestReactionFollowResolvesProjectID(t *testing.T) {
	requests := 0
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"id": float64(123)})
		case 2:
			assertRequest(t, r, "POST", "/watchers/follow.json")
			assertEqual(t, r.URL.Query().Get("target_type"), "project")
			assertEqual(t, r.URL.Query().Get("id"), "123")
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success", "watched": true})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runReactionShortcut(t, server, "follow", nil); err != nil {
		t.Fatalf("follow shortcut failed: %v", err)
	}
	assertEqual(t, requests, 2)
}

func TestReactionUnfollowUsesExplicitProjectID(t *testing.T) {
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "DELETE", "/watchers/unfollow.json")
		assertEqual(t, r.URL.Query().Get("target_type"), "project")
		assertEqual(t, r.URL.Query().Get("id"), "456")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success", "watched": false})
	})
	defer server.Close()

	if err := runReactionShortcut(t, server, "unfollow", map[string]string{"project-id": "456"}); err != nil {
		t.Fatalf("unfollow shortcut failed: %v", err)
	}
}

func TestReactionLikeUsesExplicitProjectID(t *testing.T) {
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		assertRequest(t, r, "POST", "/projects/456/praise_tread/like.json")
		writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
	})
	defer server.Close()

	if err := runReactionShortcut(t, server, "like", map[string]string{"project-id": "456"}); err != nil {
		t.Fatalf("like shortcut failed: %v", err)
	}
}

func TestReactionUnlikeResolvesStringProjectID(t *testing.T) {
	requests := 0
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		requests++
		switch requests {
		case 1:
			assertRequest(t, r, "GET", "/owner/repo.json")
			writeJSON(t, w, map[string]interface{}{"project_id": "789"})
		case 2:
			assertRequest(t, r, "DELETE", "/projects/789/praise_tread/unlike.json")
			writeJSON(t, w, map[string]interface{}{"status": 0, "message": "success"})
		default:
			t.Fatalf("unexpected extra request: %s %s", r.Method, r.URL.Path)
		}
	})
	defer server.Close()

	if err := runReactionShortcut(t, server, "unlike", nil); err != nil {
		t.Fatalf("unlike shortcut failed: %v", err)
	}
	assertEqual(t, requests, 2)
}

func TestReactionValidation(t *testing.T) {
	server := newReactionTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("invalid input should not call API, got: %s %s", r.Method, r.URL.Path)
	})
	defer server.Close()

	cases := []struct {
		name     string
		shortcut string
		args     map[string]string
	}{
		{
			name:     "invalid start",
			shortcut: "watchers",
			args:     map[string]string{"start-at": "soon"},
		},
		{
			name:     "negative end",
			shortcut: "stargazers",
			args:     map[string]string{"end-at": "-1"},
		},
		{
			name:     "invalid project id",
			shortcut: "like",
			args:     map[string]string{"project-id": "repo"},
		},
		{
			name:     "negative project id",
			shortcut: "like",
			args:     map[string]string{"project-id": "-1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := runReactionShortcut(t, server, tc.shortcut, tc.args); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func runReactionShortcut(t *testing.T, server *httptest.Server, name string, args map[string]string) error {
	t.Helper()
	shortcut := findReactionShortcut(t, name)
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

func findReactionShortcut(t *testing.T, name string) *common.Shortcut {
	t.Helper()
	for _, shortcut := range Shortcuts() {
		if shortcut.Name == name {
			return shortcut
		}
	}
	t.Fatalf("shortcut %q not found", name)
	return nil
}

func newReactionTestServer(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	return httptest.NewServer(handler)
}

func assertRequest(t *testing.T, r *http.Request, method, path string) {
	t.Helper()
	if r.Method != method || r.URL.Path != path {
		t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
	}
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
	if fmt.Sprintf("%v", got) != fmt.Sprintf("%v", want) {
		t.Fatalf("got %v (%T), want %v (%T)", got, got, want, want)
	}
}

package pr

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

const (
	checksHeadBranch = "feature/x"
	checksHeadSHA    = "82861402ada099d3e288fc41680596dde297d022"
)

type checksEnvelope struct {
	OK   bool         `json:"ok"`
	Data checksResult `json:"data"`
}

// TestPRChecksSelection mocks the single-PR GET followed by the builds GET and
// asserts which build(s) the shortcut links to the PR head under each shape of
// the (undocumented) builds payload.
func TestPRChecksSelection(t *testing.T) {
	cases := []struct {
		name          string
		builds        []interface{}
		wantMatchedBy string
		wantIDs       []float64
		wantNote      bool
	}{
		{
			name: "sha match wins over branch",
			builds: []interface{}{
				map[string]interface{}{"id": float64(1), "branch": checksHeadBranch, "sha": "deadbeef1234567", "status": "failure"},
				map[string]interface{}{"id": float64(2), "branch": checksHeadBranch, "sha": checksHeadSHA, "status": "success", "stage": "build"},
			},
			wantMatchedBy: "sha",
			wantIDs:       []float64{2},
		},
		{
			name: "abbreviated sha still matches",
			builds: []interface{}{
				map[string]interface{}{"number": float64(9), "commit_id": checksHeadSHA[:8], "state": "success"},
			},
			wantMatchedBy: "sha",
			wantIDs:       []float64{9},
		},
		{
			name: "branch fallback when no sha field present",
			builds: []interface{}{
				map[string]interface{}{"number": float64(7), "ref": "refs/heads/feature/x", "status": "running"},
				map[string]interface{}{"number": float64(8), "ref": "refs/heads/other", "status": "success"},
			},
			wantMatchedBy: "branch",
			wantIDs:       []float64{7},
		},
		{
			name: "unlinkable builds return all with a note",
			builds: []interface{}{
				map[string]interface{}{"id": float64(1), "status": "success"},
				map[string]interface{}{"id": float64(2), "status": "failure"},
			},
			wantMatchedBy: "unlinkable",
			wantIDs:       []float64{1, 2},
			wantNote:      true,
		},
		{
			name: "recognizable builds but none match the head",
			builds: []interface{}{
				map[string]interface{}{"id": float64(1), "branch": "other", "sha": "aaaaaaa1111111", "status": "success"},
			},
			wantMatchedBy: "none",
			wantIDs:       nil,
			wantNote:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == "GET" && r.URL.Path == "/owner/repo/pulls/42.json":
					writeJSON(t, w, map[string]interface{}{
						"id":              float64(42),
						"status":          "open",
						"head":            checksHeadBranch,
						"head_commit_sha": checksHeadSHA,
						"issue":           map[string]interface{}{"id": float64(100)},
					})
				case r.Method == "GET" && r.URL.Path == "/owner/repo/builds.json":
					writeJSON(t, w, tc.builds)
				default:
					t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
			}))
			defer server.Close()

			out, err := captureStdout(t, func() error {
				return runPRShortcut(t, server, "checks", map[string]string{"id": "42"})
			})
			if err != nil {
				t.Fatalf("checks failed: %v", err)
			}

			result := decodeChecksEnvelope(t, out).Data
			if result.MatchedBy != tc.wantMatchedBy {
				t.Fatalf("matched_by = %q, want %q", result.MatchedBy, tc.wantMatchedBy)
			}
			if result.HeadBranch != checksHeadBranch || result.HeadSHA != checksHeadSHA {
				t.Fatalf("head = %q/%q, want %q/%q", result.HeadBranch, result.HeadSHA, checksHeadBranch, checksHeadSHA)
			}
			if result.TotalBuilds != len(tc.builds) {
				t.Fatalf("total_builds = %d, want %d", result.TotalBuilds, len(tc.builds))
			}
			assertBuildIDs(t, result.Builds, tc.wantIDs)
			if tc.wantNote && result.Note == "" {
				t.Fatalf("expected a note for matched_by=%s", tc.wantMatchedBy)
			}
			if !tc.wantNote && result.Note != "" {
				t.Fatalf("unexpected note: %q", result.Note)
			}
		})
	}
}

func TestPRChecksErrorsWhenHeadMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/owner/repo/pulls/42.json" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		writeJSON(t, w, map[string]interface{}{"id": float64(42), "status": "open"})
	}))
	defer server.Close()

	_, err := captureStdout(t, func() error {
		return runPRShortcut(t, server, "checks", map[string]string{"id": "42"})
	})
	if err == nil {
		t.Fatal("expected error when PR response lacks head fields")
	}
}

func TestPRChecksBuildsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/owner/repo/pulls/42.json":
			writeJSON(t, w, map[string]interface{}{
				"head":            checksHeadBranch,
				"head_commit_sha": checksHeadSHA,
			})
		default:
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
		}
	}))
	defer server.Close()

	_, err := captureStdout(t, func() error {
		return runPRShortcut(t, server, "checks", map[string]string{"id": "42"})
	})
	if err == nil {
		t.Fatal("expected error when builds request fails")
	}
}

func TestExtractPullRequestHead(t *testing.T) {
	cases := []struct {
		name       string
		data       interface{}
		wantBranch string
		wantSHA    string
		wantErr    bool
	}{
		{
			name:       "top level fields",
			data:       map[string]interface{}{"head": "feature/x", "head_commit_sha": "abc123def4567"},
			wantBranch: "feature/x",
			wantSHA:    "abc123def4567",
		},
		{
			name:       "nested pull_request wrapper",
			data:       map[string]interface{}{"pull_request": map[string]interface{}{"head": "feature/y", "head_commit_sha": "def456"}},
			wantBranch: "feature/y",
			wantSHA:    "def456",
		},
		{
			name:    "missing head fields",
			data:    map[string]interface{}{"id": float64(1)},
			wantErr: true,
		},
		{
			name:    "not a map",
			data:    "raw",
			wantErr: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			branch, sha, err := extractPullRequestHead(&output.Envelope{Data: tc.data})
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if branch != tc.wantBranch || sha != tc.wantSHA {
				t.Fatalf("= %q/%q, want %q/%q", branch, sha, tc.wantBranch, tc.wantSHA)
			}
		})
	}
}

func TestBuildsFromEnvelope(t *testing.T) {
	cases := []struct {
		name string
		data interface{}
		want int
	}{
		{name: "raw json string array (client array quirk)", data: `[{"id":1},{"id":2}]`, want: 2},
		{name: "already parsed array", data: []interface{}{map[string]interface{}{"id": float64(1)}}, want: 1},
		{name: "wrapped under builds key", data: map[string]interface{}{"builds": []interface{}{map[string]interface{}{"id": float64(1)}}}, want: 1},
		{name: "non json string", data: "not json", want: 0},
		{name: "unrelated map", data: map[string]interface{}{"message": "ok"}, want: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildsFromEnvelope(&output.Envelope{Data: tc.data})
			if len(got) != tc.want {
				t.Fatalf("len = %d, want %d", len(got), tc.want)
			}
		})
	}
}

func TestCommitMatches(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"82861402ada099d3e288fc41680596dde297d022", "82861402ada099d3e288fc41680596dde297d022", true},
		{"82861402ada099d3e288fc41680596dde297d022", "8286140", true},
		{"8286140", "82861402ada099d3e288fc41680596dde297d022", true},
		{"82861402", "deadbeef", false},
		{"abc", "abc123", false},
		{"", "abc1234", false},
	}
	for _, tc := range cases {
		if got := commitMatches(tc.a, tc.b); got != tc.want {
			t.Fatalf("commitMatches(%q,%q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestSelectPullRequestChecksBranchOnlyPR(t *testing.T) {
	// A PR with only a head branch (no SHA) still links via branch.
	builds := []map[string]interface{}{
		{"id": float64(1), "branch": "feature/x", "status": "success"},
	}
	res := selectPullRequestChecks(i18n.Default(), "42", "feature/x", "", builds)
	if res.MatchedBy != "branch" || len(res.Builds) != 1 {
		t.Fatalf("matched_by=%q builds=%d, want branch/1", res.MatchedBy, len(res.Builds))
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := fn()
	w.Close()
	os.Stdout = orig
	data, readErr := io.ReadAll(r)
	if readErr != nil {
		t.Fatalf("read captured output: %v", readErr)
	}
	return string(data), runErr
}

func decodeChecksEnvelope(t *testing.T, out string) checksEnvelope {
	t.Helper()
	var env checksEnvelope
	if err := json.Unmarshal([]byte(out), &env); err != nil {
		t.Fatalf("decode output %q: %v", out, err)
	}
	return env
}

func assertBuildIDs(t *testing.T, builds []checkBuild, want []float64) {
	t.Helper()
	if len(builds) != len(want) {
		t.Fatalf("got %d builds, want %d (%v)", len(builds), len(want), want)
	}
	for i, b := range builds {
		got, ok := b.ID.(float64)
		if !ok {
			t.Fatalf("build[%d].ID = %v (%T), want float64", i, b.ID, b.ID)
		}
		if got != want[i] {
			t.Fatalf("build[%d].ID = %v, want %v", i, got, want[i])
		}
	}
}

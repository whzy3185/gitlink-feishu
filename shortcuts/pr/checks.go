package pr

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

// The GitLink builds payload is not covered by the OpenAPI reference, so the
// exact field names for a build's branch/commit/status are not guaranteed.
// We probe the conventional keys instead of hard-coding a single name the
// server may not emit, and degrade gracefully when none are present.
var (
	prHeadBranchKeys    = []string{"head", "head_branch"}
	prHeadSHAKeys       = []string{"head_commit_sha", "head_sha", "sha"}
	buildBranchKeys     = []string{"branch", "head_branch", "source_branch", "ref"}
	buildSHAKeys        = []string{"head_commit_sha", "commit_sha", "commit_id", "sha", "after", "revision"}
	buildIDKeys         = []string{"id", "number", "build_id", "build_number"}
	buildStatusKeys     = []string{"status", "state", "build_status", "phase"}
	buildStageKeys      = []string{"stage", "stage_name", "name"}
	buildConclusionKeys = []string{"conclusion", "result"}
)

type checkBuild struct {
	ID         interface{} `json:"id,omitempty"`
	Stage      string      `json:"stage,omitempty"`
	Status     string      `json:"status,omitempty"`
	Conclusion string      `json:"conclusion,omitempty"`
	Branch     string      `json:"branch,omitempty"`
	SHA        string      `json:"sha,omitempty"`
}

type checksResult struct {
	PullRequest string       `json:"pull_request"`
	HeadBranch  string       `json:"head_branch,omitempty"`
	HeadSHA     string       `json:"head_sha,omitempty"`
	MatchedBy   string       `json:"matched_by"`
	TotalBuilds int          `json:"total_builds"`
	Builds      []checkBuild `json:"builds"`
	Note        string       `json:"note,omitempty"`
}

// extractPullRequestHead reads the PR's source branch and source commit. The
// single-PR endpoint returns the PR at the top level; a nested pull_request
// object is tolerated for deployments that wrap it.
func extractPullRequestHead(env *output.Envelope) (string, string, error) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("unexpected PR response format")
	}
	branch := firstString(data, prHeadBranchKeys)
	sha := firstString(data, prHeadSHAKeys)
	if branch == "" && sha == "" {
		if nested, ok := data["pull_request"].(map[string]interface{}); ok {
			branch = firstString(nested, prHeadBranchKeys)
			sha = firstString(nested, prHeadSHAKeys)
		}
	}
	if branch == "" && sha == "" {
		return "", "", fmt.Errorf("PR response missing head branch and commit fields")
	}
	return branch, sha, nil
}

// buildsFromEnvelope normalizes the builds payload. A top-level JSON array is
// delivered by the client as a raw string (its map unmarshal fails), so string,
// array, and wrapped-object shapes all have to be handled.
func buildsFromEnvelope(env *output.Envelope) []map[string]interface{} {
	return normalizeBuildList(env.Data)
}

func normalizeBuildList(data interface{}) []map[string]interface{} {
	switch v := data.(type) {
	case string:
		var parsed interface{}
		if err := json.Unmarshal([]byte(v), &parsed); err != nil {
			return nil
		}
		return normalizeBuildList(parsed)
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]interface{}:
		for _, key := range []string{"builds", "data", "list", "items", "runs"} {
			if arr, ok := v[key].([]interface{}); ok {
				return normalizeBuildList(arr)
			}
		}
		return nil
	default:
		return nil
	}
}

// selectPullRequestChecks links CI builds to a PR head. A commit-sha match is
// authoritative; branch is the fallback. When builds expose neither field the
// linkage cannot be trusted, so every build is returned with an explanatory note.
func selectPullRequestChecks(tr *i18n.Translator, id, headBranch, headSHA string, builds []map[string]interface{}) checksResult {
	res := checksResult{
		PullRequest: id,
		HeadBranch:  headBranch,
		HeadSHA:     headSHA,
		TotalBuilds: len(builds),
		Builds:      []checkBuild{},
	}

	var shaMatches, branchMatches []checkBuild
	recognizable := false
	for _, b := range builds {
		cb := summarizeBuild(b)
		if cb.Branch != "" || cb.SHA != "" {
			recognizable = true
		}
		if headSHA != "" && cb.SHA != "" && commitMatches(cb.SHA, headSHA) {
			shaMatches = append(shaMatches, cb)
			continue
		}
		if headBranch != "" && cb.Branch != "" && cb.Branch == headBranch {
			branchMatches = append(branchMatches, cb)
		}
	}

	switch {
	case len(shaMatches) > 0:
		res.MatchedBy = "sha"
		res.Builds = shaMatches
	case len(branchMatches) > 0:
		res.MatchedBy = "branch"
		res.Builds = branchMatches
	case !recognizable && len(builds) > 0:
		res.MatchedBy = "unlinkable"
		res.Builds = summarizeBuilds(builds)
		res.Note = tr.T("output.pr.checks.unlinkable")
	default:
		res.MatchedBy = "none"
		res.Note = tr.T("output.pr.checks.no_match")
	}
	return res
}

func summarizeBuilds(builds []map[string]interface{}) []checkBuild {
	out := make([]checkBuild, 0, len(builds))
	for _, b := range builds {
		out = append(out, summarizeBuild(b))
	}
	return out
}

func summarizeBuild(b map[string]interface{}) checkBuild {
	return checkBuild{
		ID:         firstValue(b, buildIDKeys),
		Stage:      firstString(b, buildStageKeys),
		Status:     firstString(b, buildStatusKeys),
		Conclusion: firstString(b, buildConclusionKeys),
		Branch:     buildBranch(b),
		SHA:        firstString(b, buildSHAKeys),
	}
}

func buildBranch(b map[string]interface{}) string {
	for _, k := range buildBranchKeys {
		if s, ok := b[k].(string); ok && s != "" {
			return strings.TrimPrefix(s, "refs/heads/")
		}
	}
	return ""
}

// commitMatches compares two commit ids allowing an abbreviated form on either
// side, since builds may record a short SHA while the PR carries the full one.
func commitMatches(a, b string) bool {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	const minPrefix = 7
	if len(a) >= minPrefix && len(b) >= minPrefix {
		return strings.HasPrefix(a, b) || strings.HasPrefix(b, a)
	}
	return false
}

func firstString(m map[string]interface{}, keys []string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func firstValue(m map[string]interface{}, keys []string) interface{} {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

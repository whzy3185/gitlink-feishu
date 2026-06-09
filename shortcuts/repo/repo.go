package repo

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "list",
			Description: tr.T("cmd.repo.list.short"),
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: tr.T("flag.user"), Default: ""},
				{Name: "category", Short: "c", Usage: tr.T("flag.repo.category"), Default: "manage"},
				{Name: "page", Short: "p", Usage: tr.T("flag.page"), Default: "1"},
				{Name: "limit", Short: "l", Usage: tr.T("flag.limit"), Default: "20"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				user := ctx.Arg("user")
				q := url.Values{}
				q.Set("page", ctx.Arg("page"))
				q.Set("limit", ctx.Arg("limit"))
				if cat := ctx.Arg("category"); cat != "" && cat != "all" {
					q.Set("category", cat)
				}

				path := "/projects"
				if user != "" {
					path = fmt.Sprintf("/users/%s/projects", user)
				}
				env, err := ctx.CallAPIWithQuery("GET", path, q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "info",
			Description: tr.T("cmd.repo.info.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "readme",
			Description: "Show repository README content",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
				{Name: "path", Usage: "README directory path"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				if ref := ctx.Arg("ref"); ref != "" {
					q.Set("ref", ref)
				}
				if path := ctx.Arg("path"); path != "" {
					q.Set("filepath", path)
				}
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/readme", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "tree",
			Description: tr.T("cmd.repo.tree.short"),
			Flags: []common.Flag{
				{Name: "path", Short: "p", Usage: tr.T("flag.repo.tree.path")},
				{Name: "ref", Short: "r", Usage: tr.T("flag.repo.tree.ref"), Default: "master"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				q := url.Values{}
				ref := ctx.Arg("ref")
				if ref == "" {
					ref = "master"
				}
				if path := ctx.Arg("path"); path != "" {
					q.Set("filepath", path)
				}
				q.Set("ref", ref)
				env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/sub_entries", q)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "files",
			Description: "Search repository files by name",
			Flags: []common.Flag{
				{Name: "search", Short: "s", Usage: "File name keyword"},
				{Name: "ref", Short: "r", Usage: "Branch, tag, or commit SHA"},
			},
			Run: runFiles,
		},
		{
			Name:        "commit-files",
			Description: "Commit one file operation or a batch JSON file through contents/batch",
			Flags: []common.Flag{
				{Name: "branch", Short: "b", Usage: "Target branch", Required: true},
				{Name: "message", Short: "m", Usage: "Commit message", Required: true},
				{Name: "new-branch", Usage: "Create and commit to a new branch"},
				{Name: "action", Short: "a", Usage: "Single-file action: create, update, or delete; defaults to update"},
				{Name: "path", Short: "p", Usage: "Repository file path for single-file mode"},
				{Name: "content", Short: "c", Usage: "Inline file content for single-file mode"},
				{Name: "from", Usage: "Read single-file content from a local file"},
				{Name: "encoding", Usage: "Content encoding for single-file mode: text or base64; defaults to text"},
				{Name: "ops", Usage: "Read batch file operations from a JSON file"},
				{Name: "author-name", Usage: "Commit author name"},
				{Name: "author-email", Usage: "Commit author email"},
				{Name: "committer-name", Usage: "Committer name"},
				{Name: "committer-email", Usage: "Committer email"},
				{Name: "dry-run", Usage: "Preview the request without committing files", Bool: true, Default: "false"},
			},
			Run: runCommitFiles,
		},
		{
			Name:        "languages",
			Description: "Show repository language statistics",
			Run:         runLanguages,
		},
		{
			Name:        "contributors",
			Description: "List repository contributors",
			Run:         runContributors,
		},
		{
			Name:        "contributor-stats",
			Description: "List contributor statistics with code line counts",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
				{Name: "pass-year", Usage: "Number of past years to include"},
			},
			Run: runContributorStats,
		},
		{
			Name:        "code-stats",
			Description: "Show repository code statistics",
			Flags: []common.Flag{
				{Name: "ref", Usage: "Branch, tag, or commit SHA"},
			},
			Run: runCodeStats,
		},
		{
			Name:        "watchers",
			Description: "List repository watchers",
			Flags:       communityListFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runCommunityList(ctx, "watchers")
			},
		},
		{
			Name:        "stargazers",
			Description: "List repository stargazers",
			Flags:       communityListFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runCommunityList(ctx, "stargazers")
			},
		},
		{
			Name:        "follow",
			Description: "Follow a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoFollowAction(ctx, "POST", "/watchers/follow", "follow")
			},
		},
		{
			Name:        "unfollow",
			Description: "Unfollow a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoFollowAction(ctx, "DELETE", "/watchers/unfollow", "unfollow")
			},
		},
		{
			Name:        "like",
			Description: "Like a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoPraiseAction(ctx, "POST", "like")
			},
		},
		{
			Name:        "unlike",
			Description: "Unlike a repository",
			Flags:       repoInteractionFlags(),
			Run: func(ctx *common.RuntimeContext) error {
				return runRepoPraiseAction(ctx, "DELETE", "unlike")
			},
		},
		{
			Name:        "create",
			Description: tr.T("cmd.repo.create.short"),
			Flags: []common.Flag{
				{Name: "name", Short: "n", Usage: tr.T("flag.repo.name"), Required: true},
				{Name: "description", Short: "d", Usage: tr.T("flag.repo.description")},
				{Name: "private", Usage: tr.T("flag.repo.private"), Default: "false"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				name, err := ctx.RequireArg("name")
				if err != nil {
					return err
				}
				// Get current user login for the create path
				userEnv, err := ctx.CallAPI("GET", "/users/me", nil)
				if err != nil {
					return fmt.Errorf("failed to get current user: %w", err)
				}
				userData, _ := userEnv.Data.(map[string]interface{})
				login, _ := userData["login"].(string)
				if login == "" {
					return fmt.Errorf("cannot determine current user login")
				}
				userID, _ := userData["user_id"].(float64)
				body := map[string]interface{}{
					"name":            name,
					"repository_name": name,
					"user_id":         int(userID),
				}
				if desc := ctx.Arg("description"); desc != "" {
					body["description"] = desc
				}
				if ctx.Arg("private") == "true" {
					body["private"] = true
				}
				env, err := ctx.CallAPI("POST", fmt.Sprintf("/%s/%s", login, name), body)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "fork",
			Description: tr.T("cmd.repo.fork.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("POST", ctx.RepoPath()+"/forks", nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
		{
			Name:        "delete",
			Description: tr.T("cmd.repo.delete.short"),
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}
				env, err := ctx.CallAPI("DELETE", ctx.RepoPath(), nil)
				if err != nil {
					return err
				}
				return ctx.Output(env)
			},
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

type repoBatchCommitRequest struct {
	Files          []repoFileOperation `json:"files"`
	AuthorEmail    string              `json:"author_email,omitempty"`
	AuthorName     string              `json:"author_name,omitempty"`
	CommitterEmail string              `json:"committer_email,omitempty"`
	CommitterName  string              `json:"committer_name,omitempty"`
	Branch         string              `json:"branch"`
	NewBranch      string              `json:"new_branch,omitempty"`
	Message        string              `json:"message"`
}

type repoFileOperation struct {
	ActionType string  `json:"action_type"`
	Content    *string `json:"content,omitempty"`
	Encoding   string  `json:"encoding,omitempty"`
	FilePath   string  `json:"file_path"`
}

func runFiles(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "search", ctx.Arg("search"))
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/files", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommitFiles(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	req, err := buildRepoBatchCommitRequest(ctx)
	if err != nil {
		return err
	}
	if ctx.Arg("dry-run") == "true" {
		return ctx.OutputData(map[string]interface{}{
			"dry_run": true,
			"method":  "POST",
			"path":    "/v1" + ctx.RepoPath() + "/contents/batch",
			"request": req,
		})
	}
	env, err := ctx.CallAPI("POST", "/v1"+ctx.RepoPath()+"/contents/batch", req)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func buildRepoBatchCommitRequest(ctx *common.RuntimeContext) (repoBatchCommitRequest, error) {
	branch, err := requiredRepoString(ctx, "branch")
	if err != nil {
		return repoBatchCommitRequest{}, err
	}
	message, err := requiredRepoString(ctx, "message")
	if err != nil {
		return repoBatchCommitRequest{}, err
	}

	opsFile := strings.TrimSpace(ctx.Arg("ops"))
	files, err := repoCommitFileOperations(ctx, opsFile)
	if err != nil {
		return repoBatchCommitRequest{}, err
	}

	req := repoBatchCommitRequest{
		Files:          files,
		Branch:         branch,
		Message:        message,
		NewBranch:      strings.TrimSpace(ctx.Arg("new-branch")),
		AuthorName:     strings.TrimSpace(ctx.Arg("author-name")),
		AuthorEmail:    strings.TrimSpace(ctx.Arg("author-email")),
		CommitterName:  strings.TrimSpace(ctx.Arg("committer-name")),
		CommitterEmail: strings.TrimSpace(ctx.Arg("committer-email")),
	}
	if err := validateRepoIdentityPair("author", req.AuthorName, req.AuthorEmail); err != nil {
		return repoBatchCommitRequest{}, err
	}
	if err := validateRepoIdentityPair("committer", req.CommitterName, req.CommitterEmail); err != nil {
		return repoBatchCommitRequest{}, err
	}
	return req, nil
}

func repoCommitFileOperations(ctx *common.RuntimeContext, opsFile string) ([]repoFileOperation, error) {
	if opsFile != "" {
		if repoHasSingleFileArgs(ctx) {
			return nil, fmt.Errorf("--ops cannot be combined with --path, --content, --from, --action, or --encoding")
		}
		return readRepoFileOperations(opsFile)
	}
	op, err := singleRepoFileOperation(ctx)
	if err != nil {
		return nil, err
	}
	return []repoFileOperation{op}, nil
}

func repoHasSingleFileArgs(ctx *common.RuntimeContext) bool {
	for _, name := range []string{"path", "content", "from", "action", "encoding"} {
		if strings.TrimSpace(ctx.Arg(name)) != "" {
			return true
		}
	}
	return false
}

func readRepoFileOperations(path string) ([]repoFileOperation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read --ops file: %w", err)
	}
	var files []repoFileOperation
	if err := json.Unmarshal(data, &files); err == nil {
		return validateRepoFileOperations(files)
	}
	var req repoBatchCommitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("parse --ops JSON: expected an array of file operations or an object with files: %w", err)
	}
	return validateRepoFileOperations(req.Files)
}

func singleRepoFileOperation(ctx *common.RuntimeContext) (repoFileOperation, error) {
	filePath, err := requiredRepoString(ctx, "path")
	if err != nil {
		return repoFileOperation{}, err
	}
	action := strings.TrimSpace(ctx.Arg("action"))
	if action == "" {
		action = "update"
	}
	op := repoFileOperation{
		ActionType: action,
		FilePath:   filePath,
		Encoding:   strings.TrimSpace(ctx.Arg("encoding")),
	}

	hasContent := ctx.Arg("content") != ""
	fromPath := strings.TrimSpace(ctx.Arg("from"))
	hasFrom := fromPath != ""
	if hasContent && hasFrom {
		return repoFileOperation{}, fmt.Errorf("--content and --from cannot be used together")
	}
	if hasContent {
		content := ctx.Arg("content")
		op.Content = &content
	}
	if hasFrom {
		content, err := readRepoFileContent(fromPath, op.Encoding)
		if err != nil {
			return repoFileOperation{}, err
		}
		op.Content = &content
	}
	return validateRepoFileOperation(op, 0)
}

func readRepoFileContent(path, encoding string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read --from file: %w", err)
	}
	if strings.TrimSpace(encoding) == "base64" {
		return base64.StdEncoding.EncodeToString(data), nil
	}
	return string(data), nil
}

func validateRepoFileOperations(files []repoFileOperation) ([]repoFileOperation, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("at least one file operation is required")
	}
	for i := range files {
		op, err := validateRepoFileOperation(files[i], i)
		if err != nil {
			return nil, err
		}
		files[i] = op
	}
	return files, nil
}

func validateRepoFileOperation(op repoFileOperation, index int) (repoFileOperation, error) {
	prefix := fmt.Sprintf("files[%d]", index)
	op.ActionType = strings.TrimSpace(op.ActionType)
	op.FilePath = strings.TrimSpace(op.FilePath)
	op.Encoding = strings.TrimSpace(op.Encoding)

	switch op.ActionType {
	case "create", "update", "delete":
	default:
		return repoFileOperation{}, fmt.Errorf("%s.action_type must be create, update, or delete", prefix)
	}
	if op.FilePath == "" {
		return repoFileOperation{}, fmt.Errorf("%s.file_path is required", prefix)
	}
	if op.ActionType == "delete" {
		if op.Content != nil || op.Encoding != "" {
			return repoFileOperation{}, fmt.Errorf("%s delete operation must not include content or encoding", prefix)
		}
		return op, nil
	}
	if op.Content == nil {
		return repoFileOperation{}, fmt.Errorf("%s content is required for create and update operations", prefix)
	}
	if op.Encoding == "" {
		op.Encoding = "text"
	}
	if op.Encoding != "text" && op.Encoding != "base64" {
		return repoFileOperation{}, fmt.Errorf("%s.encoding must be text or base64", prefix)
	}
	return op, nil
}

func validateRepoIdentityPair(name, personName, email string) error {
	if (personName == "") != (email == "") {
		return fmt.Errorf("--%s-name and --%s-email must be provided together", name, name)
	}
	return nil
}

func requiredRepoString(ctx *common.RuntimeContext, name string) (string, error) {
	value := strings.TrimSpace(ctx.Arg(name))
	if value == "" {
		return "", fmt.Errorf("missing required flag: --%s", name)
	}
	return value, nil
}

func runLanguages(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/languages", nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runContributors(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath()+"/contributors", nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runContributorStats(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	if passYear := ctx.Arg("pass-year"); strings.TrimSpace(passYear) != "" {
		year, err := parseRepoPositiveInt(passYear, "pass-year")
		if err != nil {
			return err
		}
		q.Set("pass_year", strconv.Itoa(year))
	}
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/contributors/stat", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCodeStats(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q := url.Values{}
	setRepoQueryIfPresent(q, "ref", ctx.Arg("ref"))
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+ctx.RepoPath()+"/code_stats", q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runCommunityList(ctx *common.RuntimeContext, path string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	q, err := communityTimeRangeQuery(ctx)
	if err != nil {
		return err
	}
	env, err := ctx.CallAPIWithQuery("GET", ctx.RepoPath()+"/"+path, q)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func communityListFlags() []common.Flag {
	return []common.Flag{
		{Name: "start-at", Usage: "Start timestamp"},
		{Name: "end-at", Usage: "End timestamp"},
	}
}

func repoInteractionFlags() []common.Flag {
	return []common.Flag{
		{Name: "project-id", Usage: "GitLink project ID. If omitted, it is resolved from --owner/--repo."},
		{Name: "dry-run", Usage: "Preview the action without changing repository state", Bool: true, Default: "false"},
	}
}

func runRepoFollowAction(ctx *common.RuntimeContext, method, path, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveRepoProjectID(ctx)
	if err != nil {
		return err
	}
	query := url.Values{}
	query.Set("target_type", "project")
	query.Set("id", projectID)
	if ctx.Arg("dry-run") == "true" {
		return repoInteractionDryRun(ctx, method, path, action, projectID, query)
	}
	env, err := ctx.CallAPIWithQuery(method, path, query)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func runRepoPraiseAction(ctx *common.RuntimeContext, method, action string) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	projectID, err := resolveRepoProjectID(ctx)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/projects/%s/praise_tread/%s", projectID, action)
	if ctx.Arg("dry-run") == "true" {
		return repoInteractionDryRun(ctx, method, path, action, projectID, nil)
	}
	env, err := ctx.CallAPI(method, path, nil)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func repoInteractionDryRun(ctx *common.RuntimeContext, method, path, action, projectID string, query url.Values) error {
	preview := map[string]interface{}{
		"dry_run":    true,
		"action":     action,
		"method":     method,
		"path":       path,
		"repository": fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		"project_id": projectID,
	}
	if len(query) > 0 {
		preview["query"] = query.Encode()
	}
	return ctx.OutputData(preview)
}

func resolveRepoProjectID(ctx *common.RuntimeContext) (string, error) {
	if raw := strings.TrimSpace(ctx.Arg("project-id")); raw != "" {
		return normalizeRepoProjectID(raw)
	}
	env, err := ctx.CallAPI("GET", ctx.RepoPath(), nil)
	if err != nil {
		return "", fmt.Errorf("resolve project id: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("resolve project id: unexpected repository response")
	}
	for _, key := range []string{"id", "project_id"} {
		if id := repoProjectIDString(data[key]); id != "" {
			return id, nil
		}
	}
	return "", fmt.Errorf("resolve project id: repository response did not include id")
}

func normalizeRepoProjectID(value string) (string, error) {
	value = strings.TrimSpace(value)
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return "", fmt.Errorf("invalid --project-id %q: use a positive numeric project ID", value)
	}
	return strconv.FormatInt(parsed, 10), nil
}

func repoProjectIDString(value interface{}) string {
	switch v := value.(type) {
	case float64:
		if v > 0 {
			return strconv.FormatInt(int64(v), 10)
		}
	case int:
		if v > 0 {
			return strconv.Itoa(v)
		}
	case int64:
		if v > 0 {
			return strconv.FormatInt(v, 10)
		}
	case string:
		id, err := normalizeRepoProjectID(v)
		if err == nil {
			return id
		}
	}
	return ""
}

func communityTimeRangeQuery(ctx *common.RuntimeContext) (url.Values, error) {
	q := url.Values{}
	start, hasStart, err := parseOptionalRepoNonNegativeInt(ctx.Arg("start-at"), "start-at")
	if err != nil {
		return nil, err
	}
	end, hasEnd, err := parseOptionalRepoNonNegativeInt(ctx.Arg("end-at"), "end-at")
	if err != nil {
		return nil, err
	}
	if hasStart {
		q.Set("start_at", strconv.Itoa(start))
	}
	if hasEnd {
		q.Set("end_at", strconv.Itoa(end))
	}
	if hasStart && hasEnd && start > end {
		return nil, fmt.Errorf("--start-at must be less than or equal to --end-at")
	}
	return q, nil
}

func setRepoQueryIfPresent(q url.Values, key, value string) {
	if value := strings.TrimSpace(value); value != "" {
		q.Set(key, value)
	}
}

func parseOptionalRepoNonNegativeInt(value, name string) (int, bool, error) {
	if strings.TrimSpace(value) == "" {
		return 0, false, nil
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0, false, fmt.Errorf("invalid --%s %q: use a non-negative integer", name, value)
	}
	return parsed, true, nil
}

func parseRepoPositiveInt(value, name string) (int, error) {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("invalid --%s %q: use a positive integer", name, value)
	}
	return parsed, nil
}

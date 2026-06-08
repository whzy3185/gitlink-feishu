package pr

import (
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type prCheckoutDetail struct {
	ID           string `json:"id"`
	SourceOwner  string `json:"source_owner"`
	SourceRepo   string `json:"source_repo"`
	SourceBranch string `json:"source_branch"`
	SourceURL    string `json:"source_url"`
}

type prCheckoutPlan struct {
	PullRequest  string     `json:"pull_request"`
	SourceOwner  string     `json:"source_owner"`
	SourceRepo   string     `json:"source_repo"`
	SourceBranch string     `json:"source_branch"`
	SourceURL    string     `json:"source_url"`
	LocalBranch  string     `json:"local_branch"`
	Force        bool       `json:"force"`
	DryRun       bool       `json:"dry_run"`
	Commands     [][]string `json:"commands"`
}

type gitCommandRunner func(args ...string) (string, error)

var runGitCommand gitCommandRunner = defaultRunGitCommand

func runCheckout(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	id, err := ctx.RequireArg("id")
	if err != nil {
		return err
	}
	env, err := ctx.CallAPI("GET", prV1Path(ctx, id), nil)
	if err != nil {
		return err
	}
	detail, err := extractPRCheckoutDetail(ctx, env, id)
	if err != nil {
		return err
	}
	localBranch := strings.TrimSpace(ctx.Arg("branch"))
	if localBranch == "" {
		localBranch = "pr-" + id
	}
	force := ctx.Arg("force") == "true"
	dryRun := ctx.Arg("dry-run") == "true"
	plan, err := buildPRCheckoutPlan(detail, localBranch, force, dryRun)
	if err != nil {
		return err
	}
	if dryRun {
		return ctx.OutputData(plan)
	}
	for _, args := range plan.Commands {
		if out, err := runGitCommand(args...); err != nil {
			if strings.TrimSpace(out) != "" {
				return fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out))
			}
			return fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
		}
	}
	return ctx.OutputData(plan)
}

func buildPRCheckoutPlan(detail prCheckoutDetail, localBranch string, force, dryRun bool) (prCheckoutPlan, error) {
	if err := validateGitRef(localBranch, "local branch"); err != nil {
		return prCheckoutPlan{}, err
	}
	if err := validateGitRef(detail.SourceBranch, "source branch"); err != nil {
		return prCheckoutPlan{}, err
	}
	if detail.SourceURL == "" {
		return prCheckoutPlan{}, fmt.Errorf("PR response missing source repository URL")
	}
	checkoutFlag := "-b"
	if force {
		checkoutFlag = "-B"
	}
	return prCheckoutPlan{
		PullRequest:  detail.ID,
		SourceOwner:  detail.SourceOwner,
		SourceRepo:   detail.SourceRepo,
		SourceBranch: detail.SourceBranch,
		SourceURL:    detail.SourceURL,
		LocalBranch:  localBranch,
		Force:        force,
		DryRun:       dryRun,
		Commands: [][]string{
			{"fetch", "--no-tags", detail.SourceURL, detail.SourceBranch},
			{"checkout", checkoutFlag, localBranch, "FETCH_HEAD"},
		},
	}, nil
}

func extractPRCheckoutDetail(ctx *common.RuntimeContext, env *output.Envelope, id string) (prCheckoutDetail, error) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return prCheckoutDetail{}, fmt.Errorf("unexpected PR response format")
	}
	head := firstStringField(data, "head", "pull_request_head", "source_branch")
	if head == "" {
		if pr, ok := data["pull_request"].(map[string]interface{}); ok {
			head = firstStringField(pr, "head", "pull_request_head", "source_branch")
		}
	}
	sourceOwner, sourceRepo, sourceBranch := parsePRHeadForCheckout(head)
	fork, _ := data["fork_project"].(map[string]interface{})
	if login := stringField(fork, "login"); login != "" {
		sourceOwner = login
	}
	if identifier := stringField(fork, "identifier"); identifier != "" {
		sourceRepo = identifier
	}
	if sourceOwner == "" {
		sourceOwner = ctx.Owner
	}
	if sourceRepo == "" {
		sourceRepo = ctx.Repo
	}
	if sourceBranch == "" {
		return prCheckoutDetail{}, fmt.Errorf("PR response missing head branch")
	}
	return prCheckoutDetail{
		ID:           id,
		SourceOwner:  sourceOwner,
		SourceRepo:   sourceRepo,
		SourceBranch: sourceBranch,
		SourceURL:    gitlinkRepoURL(ctx.Client.BaseURL, sourceOwner, sourceRepo),
	}, nil
}

func parsePRHeadForCheckout(head string) (owner, repo, branch string) {
	head = strings.TrimSpace(head)
	if head == "" {
		return "", "", ""
	}
	left, right, hasOwner := strings.Cut(head, ":")
	if !hasOwner {
		return "", "", head
	}
	branch = strings.TrimSpace(right)
	parts := strings.Split(strings.Trim(left, "/"), "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], branch
	}
	if len(parts) == 1 {
		return parts[0], "", branch
	}
	return "", "", branch
}

func gitlinkRepoURL(apiBaseURL, owner, repo string) string {
	base := strings.TrimRight(apiBaseURL, "/")
	for _, suffix := range []string{"/api/v1", "/api"} {
		if strings.HasSuffix(base, suffix) {
			base = strings.TrimSuffix(base, suffix)
			break
		}
	}
	if base == "" {
		base = "https://www.gitlink.org.cn"
	}
	return fmt.Sprintf("%s/%s/%s.git", base, url.PathEscape(owner), url.PathEscape(repo))
}

func validateGitRef(ref, label string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return fmt.Errorf("%s cannot be empty", label)
	}
	if strings.HasPrefix(ref, "-") {
		return fmt.Errorf("%s cannot start with '-'", label)
	}
	if strings.ContainsAny(ref, "\x00\r\n") {
		return fmt.Errorf("%s contains unsupported control characters", label)
	}
	return nil
}

func firstStringField(m map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if v := stringField(m, key); v != "" {
			return v
		}
	}
	return ""
}

func defaultRunGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

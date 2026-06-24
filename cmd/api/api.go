package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/client"
	repoContext "github.com/gitlink-org/gitlink-cli/internal/context"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

func NewAPICmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}
	apiCmd := &cobra.Command{
		Use:   "api (<METHOD> <PATH> | --batch-file <FILE>)",
		Short: tr.T("cmd.api.short"),
		Long:  tr.T("cmd.api.long"),
		Example: `  gitlink-cli api GET /users/me
  gitlink-cli api GET /projects --query 'page=1&limit=10'
  gitlink-cli api POST /:owner/:repo/issues --body '{"subject":"Bug","description":"..."}'
  gitlink-cli api POST /:owner/:repo/issues --body-file issue.json
  gitlink-cli api POST /v1/{{owner}}/{{repo}}/issues/{{number}}/journals --body '{"notes":"handled by {{actor}}"}' --var owner=Gitlink --var repo=gitlink-cli --var number=42 --var actor=bot
  gitlink-cli api POST /v1/{{owner}}/{{repo}}/issues --body-file issue.json --var owner=Gitlink --var repo=gitlink-cli --dry-run
  gitlink-cli api --batch-file plan.json --dry-run
  gitlink-cli api --batch-file plan.json --var owner=Gitlink --var repo=gitlink-cli`,
		Args: validateAPIArgs,
		RunE: runAPI,
	}

	apiCmd.Flags().String("body", "", tr.T("flag.api.body"))
	apiCmd.Flags().String("body-file", "", tr.T("flag.api.body_file"))
	apiCmd.Flags().Bool("body-stdin", false, tr.T("flag.api.body_stdin"))
	apiCmd.Flags().String("query", "", tr.T("flag.api.query"))
	apiCmd.Flags().StringSlice("header", nil, tr.T("flag.api.header"))
	apiCmd.Flags().String("batch-file", "", tr.T("flag.api.batch_file"))
	apiCmd.Flags().Bool("dry-run", false, tr.T("flag.api.batch_dry_run"))
	apiCmd.Flags().Bool("continue-on-error", false, tr.T("flag.api.batch_continue_on_error"))
	apiCmd.Flags().StringArray("var", nil, tr.T("flag.api.batch_var"))

	return apiCmd
}

func validateAPIArgs(c *cobra.Command, args []string) error {
	batchFile, _ := c.Flags().GetString("batch-file")
	if batchFile != "" {
		if len(args) != 0 {
			return fmt.Errorf("api batch mode does not accept METHOD or PATH arguments")
		}
		return nil
	}
	return cobra.ExactArgs(2)(c, args)
}

func runAPI(c *cobra.Command, args []string) error {
	batchFile, _ := c.Flags().GetString("batch-file")
	if batchFile != "" {
		return runAPIBatch(c, batchFile)
	}

	method := strings.ToUpper(args[0])
	path := args[1]

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	vars, err := parseBatchVars(c)
	if err != nil {
		return err
	}
	addRepoContextVars(vars)
	path, err = renderSinglePath(path, vars)
	if err != nil {
		return err
	}

	cli, err := client.New()
	if err != nil {
		return err
	}
	cli.Debug = cmdutil.Debug

	body, err := readJSONBody(c, vars)
	if err != nil {
		return err
	}

	var query url.Values
	queryStr, _ := c.Flags().GetString("query")
	if queryStr != "" {
		query, err = renderSingleQuery(queryStr, vars)
		if err != nil {
			return err
		}
	}

	dryRun, _ := c.Flags().GetBool("dry-run")
	if dryRun {
		return output.Print(output.SuccessEnvelope(map[string]interface{}{
			"dry_run":   true,
			"method":    method,
			"path":      path,
			"query":     query,
			"body":      body,
			"variables": sortedVars(vars),
		}, nil), resolveFormat())
	}

	env, err := cli.Do(method, path, body, query)
	if err != nil {
		var apiErr *client.APIError
		if errors.As(err, &apiErr) {
			errEnv := output.ErrorEnvelope(apiErr.Code, apiErr.Message, "")
			return output.Print(errEnv, resolveFormat())
		}
		return err
	}

	return output.Print(env, resolveFormat())
}

func readJSONBody(c *cobra.Command, vars map[string]string) (interface{}, error) {
	bodyStr, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")
	bodyStdin, _ := c.Flags().GetBool("body-stdin")

	sources := 0
	if bodyStr != "" {
		sources++
	}
	if bodyFile != "" {
		sources++
	}
	if bodyStdin {
		sources++
	}
	if sources == 0 {
		return nil, nil
	}
	if sources > 1 {
		return nil, fmt.Errorf("use only one of --body, --body-file, or --body-stdin")
	}

	var data []byte
	var err error
	switch {
	case bodyStr != "":
		data = []byte(bodyStr)
	case bodyFile != "":
		data, err = os.ReadFile(bodyFile)
	case bodyStdin:
		data, err = io.ReadAll(c.InOrStdin())
	}
	if err != nil {
		return nil, fmt.Errorf("read JSON body: %w", err)
	}

	var body interface{}
	if err := json.Unmarshal(data, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	rendered, err := renderBatchValue(body, vars)
	if err != nil {
		return nil, fmt.Errorf("render JSON body: %w", err)
	}
	return rendered, nil
}

func renderSinglePath(path string, vars map[string]string) (string, error) {
	rendered, err := renderTemplate(rewriteColonPlaceholders(path), vars)
	if err != nil {
		return "", fmt.Errorf("render path: %w", err)
	}
	return rendered, nil
}

func renderSingleQuery(raw string, vars map[string]string) (url.Values, error) {
	rendered, err := renderTemplate(raw, vars)
	if err != nil {
		return nil, fmt.Errorf("render query: %w", err)
	}
	query, err := url.ParseQuery(rendered)
	if err != nil {
		return nil, fmt.Errorf("invalid query string: %w", err)
	}
	return query, nil
}

func addRepoContextVars(vars map[string]string) {
	if vars == nil {
		return
	}
	if vars["owner"] != "" && vars["repo"] != "" {
		return
	}
	owner, repo, err := repoContext.ResolveOwnerRepo(cmdutil.Owner, cmdutil.Repo)
	if err != nil {
		return
	}
	if vars["owner"] == "" {
		vars["owner"] = owner
	}
	if vars["repo"] == "" {
		vars["repo"] = repo
	}
}

func rewriteColonPlaceholders(path string) string {
	return colonPathVarPattern.ReplaceAllString(path, `$1{{$2}}`)
}

var colonPathVarPattern = templatePattern

func init() {
	colonPathVarPattern = mustCompileColonPattern()
}

func mustCompileColonPattern() *regexp.Regexp {
	return regexp.MustCompile(`(^|/):([A-Za-z0-9_.-]+)`)
}

func resolveFormat() string {
	f := cmdutil.Format
	if f == "" {
		return "json"
	}
	return f
}

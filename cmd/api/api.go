package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/client"
	gitcontext "github.com/gitlink-org/gitlink-cli/internal/context"
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
  gitlink-cli api GET /{{owner}}/{{repo}}/pulls --var owner=Gitlink --var repo=gitlink-cli
  gitlink-cli api POST /:owner/:repo/issues --body-file issue.json
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

	body, err := readJSONBody(c)
	if err != nil {
		return err
	}

	var query url.Values
	queryStr, _ := c.Flags().GetString("query")
	if queryStr != "" {
		var err error
		query, err = url.ParseQuery(queryStr)
		if err != nil {
			return fmt.Errorf("invalid query string: %w", err)
		}
	}

	headers, err := parseAPIHeaders(c)
	if err != nil {
		return err
	}

	request, err := renderSingleAPIRequest(args[1], query, body, headers, c)
	if err != nil {
		return err
	}

	cli, err := client.New()
	if err != nil {
		return err
	}
	cli.Debug = cmdutil.Debug

	env, err := cli.DoWithHeaders(method, request.Path, request.Body, request.Query, request.Headers)
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

func readJSONBody(c *cobra.Command) (interface{}, error) {
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
	return body, nil
}

func resolveFormat() string {
	f := cmdutil.Format
	if f == "" {
		return "json"
	}
	return f
}

type singleAPIRequest struct {
	Path    string
	Query   url.Values
	Body    interface{}
	Headers http.Header
}

func renderSingleAPIRequest(path string, query url.Values, body interface{}, headers http.Header, c *cobra.Command) (*singleAPIRequest, error) {
	normalizedPath := normalizeSingleAPIPath(path)
	if !strings.HasPrefix(normalizedPath, "/") {
		normalizedPath = "/" + normalizedPath
	}

	vars, err := resolveSingleRequestVars(c, normalizedPath, query, body, headers)
	if err != nil {
		return nil, err
	}

	renderedPath, err := renderTemplate(normalizedPath, vars)
	if err != nil {
		return nil, fmt.Errorf("render path: %w", err)
	}
	renderedQuery, err := renderURLValues(query, vars)
	if err != nil {
		return nil, fmt.Errorf("render query: %w", err)
	}
	renderedBody, err := renderBatchValue(body, vars)
	if err != nil {
		return nil, fmt.Errorf("render body: %w", err)
	}
	renderedHeaders, err := renderAPIHeaders(headers, vars)
	if err != nil {
		return nil, fmt.Errorf("render headers: %w", err)
	}

	return &singleAPIRequest{
		Path:    renderedPath,
		Query:   renderedQuery,
		Body:    renderedBody,
		Headers: renderedHeaders,
	}, nil
}

func normalizeSingleAPIPath(path string) string {
	return strings.NewReplacer(":owner", "{{owner}}", ":repo", "{{repo}}").Replace(strings.TrimSpace(path))
}

func resolveSingleRequestVars(c *cobra.Command, path string, query url.Values, body interface{}, headers http.Header) (map[string]string, error) {
	vars, err := parseBatchVars(c)
	if err != nil {
		return nil, err
	}
	if !needsOwnerRepoResolution(path, query, body, headers, vars) {
		return vars, nil
	}
	owner, repo, err := gitcontext.ResolveOwnerRepo(cmdutil.Owner, cmdutil.Repo)
	if err != nil {
		return nil, fmt.Errorf("resolve owner/repo for api templates: %w", err)
	}
	if _, ok := vars["owner"]; !ok {
		vars["owner"] = owner
	}
	if _, ok := vars["repo"]; !ok {
		vars["repo"] = repo
	}
	return vars, nil
}

func needsOwnerRepoResolution(path string, query url.Values, body interface{}, headers http.Header, vars map[string]string) bool {
	if vars["owner"] != "" && vars["repo"] != "" {
		return false
	}
	if strings.Contains(path, "{{owner}}") || strings.Contains(path, "{{repo}}") {
		return true
	}
	for key, values := range query {
		if strings.Contains(key, "{{owner}}") || strings.Contains(key, "{{repo}}") {
			return true
		}
		for _, value := range values {
			if strings.Contains(value, "{{owner}}") || strings.Contains(value, "{{repo}}") {
				return true
			}
		}
	}
	if containsTemplateVar(body, "owner", "repo") {
		return true
	}
	for key, values := range headers {
		if strings.Contains(key, "{{owner}}") || strings.Contains(key, "{{repo}}") {
			return true
		}
		for _, value := range values {
			if strings.Contains(value, "{{owner}}") || strings.Contains(value, "{{repo}}") {
				return true
			}
		}
	}
	return false
}

func containsTemplateVar(value interface{}, names ...string) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case string:
		for _, name := range names {
			if strings.Contains(typed, "{{"+name+"}}") {
				return true
			}
		}
		return false
	case []interface{}:
		for _, item := range typed {
			if containsTemplateVar(item, names...) {
				return true
			}
		}
		return false
	case map[string]interface{}:
		for key, item := range typed {
			if containsTemplateVar(key, names...) || containsTemplateVar(item, names...) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func parseAPIHeaders(c *cobra.Command) (http.Header, error) {
	rawHeaders, _ := c.Flags().GetStringSlice("header")
	if len(rawHeaders) == 0 {
		return nil, nil
	}
	headers := http.Header{}
	for _, item := range rawHeaders {
		name, value, ok := strings.Cut(item, ":")
		if !ok {
			return nil, fmt.Errorf("invalid --header %q, want key:value", item)
		}
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" {
			return nil, fmt.Errorf("invalid --header %q, header name cannot be empty", item)
		}
		headers.Add(name, value)
	}
	return headers, nil
}

func renderAPIHeaders(headers http.Header, vars map[string]string) (http.Header, error) {
	if len(headers) == 0 {
		return nil, nil
	}
	rendered := http.Header{}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		renderedKey, err := renderTemplate(key, vars)
		if err != nil {
			return nil, err
		}
		for _, value := range headers.Values(key) {
			renderedValue, err := renderTemplate(value, vars)
			if err != nil {
				return nil, err
			}
			rendered.Add(renderedKey, renderedValue)
		}
	}
	return rendered, nil
}

func renderURLValues(query url.Values, vars map[string]string) (url.Values, error) {
	if len(query) == 0 {
		return nil, nil
	}
	rendered := url.Values{}
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		renderedKey, err := renderTemplate(key, vars)
		if err != nil {
			return nil, err
		}
		for _, value := range query[key] {
			renderedValue, err := renderTemplate(value, vars)
			if err != nil {
				return nil, err
			}
			rendered.Add(renderedKey, renderedValue)
		}
	}
	return rendered, nil
}

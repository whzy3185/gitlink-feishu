package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/client"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

type batchPlan struct {
	Vars     map[string]string `json:"vars"`
	Requests []batchRequest    `json:"requests"`
}

type batchRequest struct {
	Name   string                 `json:"name"`
	Method string                 `json:"method"`
	Path   string                 `json:"path"`
	Query  map[string]interface{} `json:"query"`
	Body   interface{}            `json:"body"`
}

type renderedBatchRequest struct {
	Index  int         `json:"index" yaml:"index"`
	Name   string      `json:"name,omitempty" yaml:"name,omitempty"`
	Method string      `json:"method" yaml:"method"`
	Path   string      `json:"path" yaml:"path"`
	Query  url.Values  `json:"query,omitempty" yaml:"query,omitempty"`
	Body   interface{} `json:"body,omitempty" yaml:"body,omitempty"`
}

type batchResult struct {
	Index  int         `json:"index" yaml:"index"`
	Name   string      `json:"name,omitempty" yaml:"name,omitempty"`
	Method string      `json:"method" yaml:"method"`
	Path   string      `json:"path" yaml:"path"`
	OK     bool        `json:"ok" yaml:"ok"`
	Error  string      `json:"error,omitempty" yaml:"error,omitempty"`
	Data   interface{} `json:"data,omitempty" yaml:"data,omitempty"`
}

type batchSummary struct {
	DryRun          bool                   `json:"dry_run" yaml:"dry_run"`
	ContinueOnError bool                   `json:"continue_on_error" yaml:"continue_on_error"`
	Total           int                    `json:"total" yaml:"total"`
	Succeeded       int                    `json:"succeeded" yaml:"succeeded"`
	Failed          int                    `json:"failed" yaml:"failed"`
	Variables       map[string]string      `json:"variables,omitempty" yaml:"variables,omitempty"`
	Requests        []renderedBatchRequest `json:"requests,omitempty" yaml:"requests,omitempty"`
	Results         []batchResult          `json:"results,omitempty" yaml:"results,omitempty"`
}

var templatePattern = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.-]+)\s*\}\}`)

func runAPIBatch(c *cobra.Command, batchFile string) error {
	if hasSingleRequestInput(c) {
		return fmt.Errorf("use batch flags separately from --body, --body-file, --body-stdin, --query, or --header")
	}

	dryRun, _ := c.Flags().GetBool("dry-run")
	continueOnError, _ := c.Flags().GetBool("continue-on-error")
	overrides, err := parseBatchVars(c)
	if err != nil {
		return err
	}
	plan, err := readBatchPlan(batchFile)
	if err != nil {
		return err
	}

	vars := mergeBatchVars(plan.Vars, overrides)
	requests, err := renderBatchRequests(plan.Requests, vars)
	if err != nil {
		return err
	}
	if dryRun {
		return output.Print(output.SuccessEnvelope(batchSummary{
			DryRun:          true,
			ContinueOnError: continueOnError,
			Total:           len(requests),
			Variables:       sortedVars(vars),
			Requests:        requests,
		}, nil), resolveFormat())
	}

	cli, err := client.New()
	if err != nil {
		return err
	}
	cli.Debug = cmdutil.Debug

	summary := batchSummary{
		DryRun:          false,
		ContinueOnError: continueOnError,
		Total:           len(requests),
		Variables:       sortedVars(vars),
		Results:         make([]batchResult, 0, len(requests)),
	}
	for _, req := range requests {
		result := batchResult{
			Index:  req.Index,
			Name:   req.Name,
			Method: req.Method,
			Path:   req.Path,
		}
		env, callErr := cli.Do(req.Method, req.Path, req.Body, req.Query)
		if callErr != nil {
			summary.Failed++
			result.OK = false
			result.Error = apiBatchErrorMessage(callErr)
			summary.Results = append(summary.Results, result)
			if !continueOnError {
				_ = output.Print(output.SuccessEnvelope(summary, nil), resolveFormat())
				return callErr
			}
			continue
		}
		summary.Succeeded++
		result.OK = true
		if env != nil {
			result.Data = env.Data
		}
		summary.Results = append(summary.Results, result)
	}

	return output.Print(output.SuccessEnvelope(summary, nil), resolveFormat())
}

func hasSingleRequestInput(c *cobra.Command) bool {
	body, _ := c.Flags().GetString("body")
	bodyFile, _ := c.Flags().GetString("body-file")
	bodyStdin, _ := c.Flags().GetBool("body-stdin")
	query, _ := c.Flags().GetString("query")
	headers, _ := c.Flags().GetStringSlice("header")
	return body != "" || bodyFile != "" || bodyStdin || query != "" || len(headers) > 0
}

func parseBatchVars(c *cobra.Command) (map[string]string, error) {
	raw, _ := c.Flags().GetStringArray("var")
	vars := make(map[string]string, len(raw))
	for _, item := range raw {
		key, value, ok := strings.Cut(item, "=")
		key = strings.TrimSpace(key)
		if !ok || key == "" {
			return nil, fmt.Errorf("invalid --var %q, want key=value", item)
		}
		vars[key] = value
	}
	return vars, nil
}

func readBatchPlan(path string) (*batchPlan, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read batch file: %w", err)
	}
	var plan batchPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("invalid batch file JSON: %w", err)
	}
	if len(plan.Requests) == 0 {
		return nil, fmt.Errorf("batch file must contain at least one request")
	}
	return &plan, nil
}

func mergeBatchVars(planVars, overrides map[string]string) map[string]string {
	vars := make(map[string]string, len(planVars)+len(overrides))
	for key, value := range planVars {
		vars[key] = value
	}
	for key, value := range overrides {
		vars[key] = value
	}
	return vars
}

func renderBatchRequests(requests []batchRequest, vars map[string]string) ([]renderedBatchRequest, error) {
	rendered := make([]renderedBatchRequest, 0, len(requests))
	for i, req := range requests {
		method := strings.ToUpper(strings.TrimSpace(req.Method))
		if method == "" {
			return nil, fmt.Errorf("request %d method is required", i+1)
		}
		path, err := renderTemplate(req.Path, vars)
		if err != nil {
			return nil, fmt.Errorf("request %d path: %w", i+1, err)
		}
		path = strings.TrimSpace(path)
		if path == "" {
			return nil, fmt.Errorf("request %d path is required", i+1)
		}
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
		query, err := renderBatchQuery(req.Query, vars)
		if err != nil {
			return nil, fmt.Errorf("request %d query: %w", i+1, err)
		}
		body, err := renderBatchValue(req.Body, vars)
		if err != nil {
			return nil, fmt.Errorf("request %d body: %w", i+1, err)
		}
		name, err := renderTemplate(req.Name, vars)
		if err != nil {
			return nil, fmt.Errorf("request %d name: %w", i+1, err)
		}
		rendered = append(rendered, renderedBatchRequest{
			Index:  i + 1,
			Name:   name,
			Method: method,
			Path:   path,
			Query:  query,
			Body:   body,
		})
	}
	return rendered, nil
}

func renderBatchQuery(raw map[string]interface{}, vars map[string]string) (url.Values, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	query := url.Values{}
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		renderedKey, err := renderTemplate(key, vars)
		if err != nil {
			return nil, err
		}
		values, err := renderQueryValues(raw[key], vars)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		for _, value := range values {
			query.Add(renderedKey, value)
		}
	}
	return query, nil
}

func renderQueryValues(raw interface{}, vars map[string]string) ([]string, error) {
	switch value := raw.(type) {
	case nil:
		return []string{""}, nil
	case string:
		rendered, err := renderTemplate(value, vars)
		return []string{rendered}, err
	case []interface{}:
		values := make([]string, 0, len(value))
		for _, item := range value {
			itemValues, err := renderQueryValues(item, vars)
			if err != nil {
				return nil, err
			}
			values = append(values, itemValues...)
		}
		return values, nil
	default:
		return []string{fmt.Sprint(value)}, nil
	}
}

func renderBatchValue(raw interface{}, vars map[string]string) (interface{}, error) {
	switch value := raw.(type) {
	case nil:
		return nil, nil
	case string:
		return renderTemplate(value, vars)
	case []interface{}:
		items := make([]interface{}, 0, len(value))
		for _, item := range value {
			rendered, err := renderBatchValue(item, vars)
			if err != nil {
				return nil, err
			}
			items = append(items, rendered)
		}
		return items, nil
	case map[string]interface{}:
		obj := make(map[string]interface{}, len(value))
		for key, item := range value {
			renderedKey, err := renderTemplate(key, vars)
			if err != nil {
				return nil, err
			}
			rendered, err := renderBatchValue(item, vars)
			if err != nil {
				return nil, err
			}
			obj[renderedKey] = rendered
		}
		return obj, nil
	default:
		return raw, nil
	}
}

func renderTemplate(value string, vars map[string]string) (string, error) {
	var missing []string
	rendered := templatePattern.ReplaceAllStringFunc(value, func(match string) string {
		parts := templatePattern.FindStringSubmatch(match)
		if len(parts) != 2 {
			return match
		}
		replacement, ok := vars[parts[1]]
		if !ok {
			missing = append(missing, parts[1])
			return match
		}
		return replacement
	})
	if len(missing) > 0 {
		sort.Strings(missing)
		return "", fmt.Errorf("missing template variable(s): %s", strings.Join(missing, ", "))
	}
	return rendered, nil
}

func sortedVars(vars map[string]string) map[string]string {
	if len(vars) == 0 {
		return nil
	}
	copyVars := make(map[string]string, len(vars))
	for key, value := range vars {
		copyVars[key] = value
	}
	return copyVars
}

func apiBatchErrorMessage(err error) string {
	var apiErr *client.APIError
	if errors.As(err, &apiErr) {
		return apiErr.Message
	}
	return err.Error()
}

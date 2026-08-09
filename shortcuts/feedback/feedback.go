package feedback

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

var feedbackInput io.Reader = os.Stdin

// Shortcuts returns GitLink platform feedback shortcuts.
func Shortcuts() []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "create",
			Description: "Submit feedback or suggestions to GitLink",
			Flags: []common.Flag{
				{Name: "user", Short: "u", Usage: "GitLink user login. Defaults to current authenticated user"},
				{Name: "content", Short: "c", Usage: "Feedback content"},
				{Name: "from", Short: "f", Usage: "Read feedback content from a text file"},
				{Name: "stdin", Usage: "Read feedback content from standard input", Bool: true, Default: "false"},
				{Name: "category", Usage: "Optional feedback category, for example bug, feature, docs, ux, or cli"},
				{Name: "contact", Usage: "Optional contact information to include in the feedback"},
				{Name: "repo-ref", Usage: "Optional related repository in owner/repo form"},
				{Name: "dry-run", Usage: "Preview the request without submitting feedback", Bool: true, Default: "false"},
			},
			Run: runCreate,
		},
	}
}

func runCreate(ctx *common.RuntimeContext) error {
	user, err := resolveFeedbackUser(ctx)
	if err != nil {
		return err
	}
	content, err := buildFeedbackContent(ctx)
	if err != nil {
		return err
	}
	payload := map[string]interface{}{"content": content}
	path := feedbackPath(user)
	if parseFeedbackBool(ctx.Arg("dry-run")) {
		return ctx.OutputData(map[string]interface{}{
			"dry_run":        true,
			"method":         "POST",
			"path":           path,
			"user":           user,
			"content_length": len(content),
			"body":           payload,
		})
	}
	env, err := ctx.CallAPI("POST", path, payload)
	if err != nil {
		return err
	}
	return ctx.Output(env)
}

func resolveFeedbackUser(ctx *common.RuntimeContext) (string, error) {
	if user := strings.TrimSpace(ctx.Arg("user")); user != "" {
		return user, nil
	}
	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("cannot determine current user login")
	}
	login, _ := data["login"].(string)
	login = strings.TrimSpace(login)
	if login == "" {
		return "", fmt.Errorf("cannot determine current user login")
	}
	return login, nil
}

func buildFeedbackContent(ctx *common.RuntimeContext) (string, error) {
	parts := make([]string, 0, 3)
	if content := strings.TrimSpace(ctx.Arg("content")); content != "" {
		parts = append(parts, content)
	}
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		content, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read --from: %w", err)
		}
		if text := strings.TrimSpace(string(content)); text != "" {
			parts = append(parts, text)
		}
	}
	if parseFeedbackBool(ctx.Arg("stdin")) {
		content, err := io.ReadAll(feedbackInput)
		if err != nil {
			return "", fmt.Errorf("read --stdin: %w", err)
		}
		if text := strings.TrimSpace(string(content)); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("provide feedback content with --content, --from, or --stdin")
	}

	metadata := feedbackMetadata(ctx)
	body := strings.Join(parts, "\n\n")
	if len(metadata) == 0 {
		return body, nil
	}
	return strings.Join(append(metadata, "", body), "\n"), nil
}

func feedbackMetadata(ctx *common.RuntimeContext) []string {
	var lines []string
	if category := strings.TrimSpace(ctx.Arg("category")); category != "" {
		lines = append(lines, "Category: "+category)
	}
	if repoRef := strings.TrimSpace(ctx.Arg("repo-ref")); repoRef != "" {
		lines = append(lines, "Repository: "+repoRef)
	} else if strings.TrimSpace(ctx.Owner) != "" && strings.TrimSpace(ctx.Repo) != "" {
		lines = append(lines, fmt.Sprintf("Repository: %s/%s", strings.TrimSpace(ctx.Owner), strings.TrimSpace(ctx.Repo)))
	}
	if contact := strings.TrimSpace(ctx.Arg("contact")); contact != "" {
		lines = append(lines, "Contact: "+contact)
	}
	return lines
}

func feedbackPath(user string) string {
	return fmt.Sprintf("/v1/%s/feedbacks", url.PathEscape(user))
}

func parseFeedbackBool(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

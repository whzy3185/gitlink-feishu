package browse

import (
	"bytes"
	"strings"
	"testing"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/web"
)

func TestNewBrowseCmd(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.Use != "browse [resource]" {
		t.Errorf("expected Use 'browse [resource]', got %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestBrowseCmdHasCorrectArgs(t *testing.T) {
	cmd := NewBrowseCmd()
	if err := cmd.Args(cmd, []string{}); err != nil {
		t.Errorf("should accept 0 args: %v", err)
	}
	if err := cmd.Args(cmd, []string{"issues/42"}); err != nil {
		t.Errorf("should accept 1 arg: %v", err)
	}
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("should reject more than 1 arg")
	}
}

func TestBrowseCmdNoSubcommands(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.HasSubCommands() {
		t.Error("browse should not have subcommands")
	}
}

func TestBrowseCmdExample(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.Example == "" {
		t.Error("Example should not be empty")
	}
	if !strings.Contains(cmd.Example, "browse") {
		t.Error("Example should contain 'browse'")
	}
}

func TestBrowseCmdHasListAndNoOpenFlags(t *testing.T) {
	cmd := NewBrowseCmd()
	if cmd.Flags().Lookup("list") == nil {
		t.Error("missing --list flag")
	}
	if cmd.Flags().Lookup("no-open") == nil {
		t.Error("missing --no-open flag")
	}
}

func TestResolveBrowseURL(t *testing.T) {
	b := web.NewBuilder()
	cases := []struct {
		name    string
		args    []string
		wantSub string
	}{
		{"no args → repo", nil, "/o/r"},
		{"issue detail", []string{"issues/42"}, "/issues/42"},
		{"issue alias", []string{"issue/7"}, "/issues/7"},
		{"pr detail", []string{"pulls/128"}, "/pulls/128"},
		{"pr alias", []string{"pr/9"}, "/pulls/9"},
		{"wiki index", []string{"wiki"}, "/wiki"},
		{"wiki page", []string{"wiki/Guide"}, "/wiki/Guide"},
		{"ci", []string{"actions"}, "/actions"},
		{"ci alias", []string{"ci"}, "/actions"},
		{"commit", []string{"commits/abc123"}, "/commits/abc123"},
		{"release", []string{"releases/v2.0"}, "/releases/v2.0"},
		{"milestones", []string{"milestones"}, "/milestones"},
		{"labels", []string{"labels"}, "/issues/labels"},
		{"webhook settings", []string{"settings/hooks"}, "/settings/hooks"},
		{"collaboration", []string{"settings/collaboration"}, "/settings/collaboration"},
		{"unknown passthrough", []string{"custom/seg"}, "/custom/seg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := resolveBrowseURL(b, "o", "r", c.args)
			if !strings.Contains(r.URL, c.wantSub) {
				t.Errorf("URL %q missing %q", r.URL, c.wantSub)
			}
		})
	}
}

func TestBrowseListOutputsCatalog(t *testing.T) {
	out := runBrowse(t, "--owner", "o", "--repo", "r", "--list")
	for _, want := range []string{"issues", "pulls", "wiki", "actions"} {
		if !strings.Contains(out, want) {
			t.Errorf("list missing %q: %q", want, out)
		}
	}
}

func TestBrowseJSONFormat(t *testing.T) {
	// --format json must route emitBrowse through the output envelope.
	out := runBrowseFmt(t, "json", "issues/42")
	if !strings.Contains(out, `"html_url"`) {
		t.Errorf("JSON browse missing html_url: %q", out)
	}
}

func TestResolveBrowseURLIssueNonNumeric(t *testing.T) {
	// atoiOrZero("abc") should fall back to 0 (issue list page).
	r := resolveBrowseURL(web.NewBuilder(), "o", "r", []string{"issues/abc"})
	if !strings.HasSuffix(r.URL, "/issues") {
		t.Errorf("expected /issues fallback, got %q", r.URL)
	}
}

func TestBrowseNoOpenDoesNotLaunchBrowser(t *testing.T) {
	// --no-open must print the URL but never invoke a browser. We can't easily
	// stub web.OpenBrowser across packages, so we assert the URL is printed
	// and that the "browser did not open" warning (printed only when
	// OpenBrowser returns an error) is absent.
	out := runBrowseNoOpen(t, "issues/42", true)
	if !strings.Contains(out, "/issues/42") {
		t.Errorf("expected /issues/42 in output: %q", out)
	}
	if strings.Contains(out, "浏览器未自动打开") {
		t.Errorf("--no-open should not print open-failure warning: %q", out)
	}
}

// runBrowse runs `browse <args>` with captured stdout.
func runBrowse(t *testing.T, args ...string) string {
	t.Helper()
	old := stdout
	oldOwner, oldRepo, oldFmt := cmdutil.Owner, cmdutil.Repo, cmdutil.Format
	buf := &bytes.Buffer{}
	stdout = buf
	defer func() {
		stdout = old
		cmdutil.Owner, cmdutil.Repo, cmdutil.Format = oldOwner, oldRepo, oldFmt
	}()
	cmd := NewBrowseCmd()
	cmd.PersistentFlags().StringVar(&cmdutil.Owner, "owner", "", "")
	cmd.PersistentFlags().StringVar(&cmdutil.Repo, "repo", "", "")
	cmd.PersistentFlags().StringVar(&cmdutil.Format, "format", "", "")
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("browse %v: %v", args, err)
	}
	return buf.String()
}

// runBrowseFmt runs `browse <resource>` with a specific --format value.
func runBrowseFmt(t *testing.T, format, resource string) string {
	t.Helper()
	return runBrowse(t, "--owner", "o", "--repo", "r", "--format", format, "--no-open", resource)
}

func runBrowseNoOpen(t *testing.T, resource string, noOpen bool) string {
	t.Helper()
	args := []string{"--owner", "o", "--repo", "r"}
	if resource != "" {
		args = append(args, resource)
	}
	if noOpen {
		args = append(args, "--no-open")
	}
	return runBrowse(t, args...)
}

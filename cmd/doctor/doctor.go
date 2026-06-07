package doctor

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	internalAuth "github.com/gitlink-org/gitlink-cli/internal/auth"
	internalConfig "github.com/gitlink-org/gitlink-cli/internal/config"
	repoContext "github.com/gitlink-org/gitlink-cli/internal/context"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/internal/output"
)

const (
	statusOK      = "ok"
	statusWarning = "warning"
	statusError   = "error"
	statusSkipped = "skipped"
)

var (
	loadConfig       = internalConfig.Load
	loadToken        = internalAuth.LoadToken
	getCurrentUser   = internalAuth.GetCurrentUser
	resolveOwnerRepo = repoContext.ResolveOwnerRepo
	statFile         = os.Stat
	lookupEnv        = os.LookupEnv
)

type Report struct {
	OK      bool     `json:"ok"`
	Summary Summary  `json:"summary"`
	Checks  []Check  `json:"checks"`
	Actions []string `json:"actions,omitempty"`
}

type Summary struct {
	OK      int `json:"ok"`
	Warning int `json:"warning"`
	Error   int `json:"error"`
	Skipped int `json:"skipped"`
	Total   int `json:"total"`
}

type Check struct {
	Name       string                 `json:"name"`
	Status     string                 `json:"status"`
	Message    string                 `json:"message"`
	Suggestion string                 `json:"suggestion,omitempty"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

func NewDoctorCmd(translators ...*i18n.Translator) *cobra.Command {
	tr := i18n.Default()
	if len(translators) > 0 && translators[0] != nil {
		tr = translators[0]
	}

	var skipNetwork bool
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: tr.T("cmd.doctor.short"),
		Long:  tr.T("cmd.doctor.long"),
		RunE: func(cmd *cobra.Command, args []string) error {
			report := Run(skipNetwork, tr)
			return output.PrintTo(cmd.OutOrStdout(), output.SuccessEnvelope(report, nil), resolveFormat())
		},
	}
	cmd.Flags().BoolVar(&skipNetwork, "skip-network", false, tr.T("flag.doctor.skip_network"))
	return cmd
}

func Run(skipNetwork bool, tr *i18n.Translator) Report {
	if tr == nil {
		tr = i18n.Default()
	}

	checks := make([]Check, 0, 5)
	cfg, cfgErr := loadConfig()
	checks = append(checks, checkConfigFile(tr, cfgErr))
	checks = append(checks, checkConfigValues(tr, cfg, cfgErr))
	checks = append(checks, checkAuthToken(tr))
	checks = append(checks, checkRepoContext(tr))
	checks = append(checks, checkAuthenticatedUser(tr, skipNetwork, cfgErr))

	report := Report{OK: true, Checks: checks}
	seenActions := map[string]bool{}
	for _, check := range checks {
		report.Summary.Total++
		switch check.Status {
		case statusOK:
			report.Summary.OK++
		case statusWarning:
			report.Summary.Warning++
		case statusError:
			report.OK = false
			report.Summary.Error++
		case statusSkipped:
			report.Summary.Skipped++
		}
		if check.Suggestion != "" && !seenActions[check.Suggestion] {
			report.Actions = append(report.Actions, check.Suggestion)
			seenActions[check.Suggestion] = true
		}
	}
	return report
}

func checkConfigFile(tr *i18n.Translator, cfgErr error) Check {
	path := internalConfig.ConfigPath()
	info, err := statFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Check{
				Name:       "config_file",
				Status:     statusWarning,
				Message:    tr.T("output.doctor.config_file.missing"),
				Suggestion: "gitlink-cli config init",
				Details:    map[string]interface{}{"path": path},
			}
		}
		return Check{
			Name:       "config_file",
			Status:     statusError,
			Message:    tr.Tf("output.doctor.config_file.unreadable", i18n.Args{"message": err.Error()}),
			Suggestion: tr.T("output.doctor.suggestion.check_config_permissions"),
			Details:    map[string]interface{}{"path": path},
		}
	}
	if cfgErr != nil {
		return Check{
			Name:       "config_file",
			Status:     statusError,
			Message:    tr.Tf("output.doctor.config_file.invalid", i18n.Args{"message": cfgErr.Error()}),
			Suggestion: tr.T("output.doctor.suggestion.fix_config_yaml"),
			Details:    map[string]interface{}{"path": path},
		}
	}
	return Check{
		Name:    "config_file",
		Status:  statusOK,
		Message: tr.T("output.doctor.config_file.ok"),
		Details: map[string]interface{}{
			"path": path,
			"size": info.Size(),
		},
	}
}

func checkConfigValues(tr *i18n.Translator, cfg *internalConfig.Config, cfgErr error) Check {
	if cfgErr != nil || cfg == nil {
		return Check{
			Name:       "config_values",
			Status:     statusSkipped,
			Message:    tr.T("output.doctor.config_values.skipped"),
			Suggestion: tr.T("output.doctor.suggestion.fix_config_yaml"),
		}
	}

	details := map[string]interface{}{
		"base_url":       cfg.BaseURL,
		"default_format": cfg.Format,
	}
	if err := validateBaseURL(cfg.BaseURL); err != nil {
		return Check{
			Name:       "config_values",
			Status:     statusError,
			Message:    tr.Tf("output.doctor.config_values.bad_base_url", i18n.Args{"message": err.Error()}),
			Suggestion: "gitlink-cli config set base_url https://www.gitlink.org.cn/api",
			Details:    details,
		}
	}
	if !validFormat(cfg.Format) {
		return Check{
			Name:       "config_values",
			Status:     statusWarning,
			Message:    tr.Tf("output.doctor.config_values.bad_format", i18n.Args{"format": cfg.Format}),
			Suggestion: "gitlink-cli config set default_format table",
			Details:    details,
		}
	}
	return Check{
		Name:    "config_values",
		Status:  statusOK,
		Message: tr.T("output.doctor.config_values.ok"),
		Details: details,
	}
}

func checkAuthToken(tr *i18n.Translator) Check {
	if token, ok := lookupEnv("GITLINK_TOKEN"); ok && strings.TrimSpace(token) != "" {
		return Check{
			Name:    "auth_token",
			Status:  statusOK,
			Message: tr.T("output.doctor.auth_token.env"),
			Details: map[string]interface{}{"source": "env"},
		}
	}
	token, err := loadToken()
	if err != nil || strings.TrimSpace(token) == "" {
		return Check{
			Name:       "auth_token",
			Status:     statusWarning,
			Message:    tr.T("output.doctor.auth_token.missing"),
			Suggestion: "gitlink-cli auth login",
		}
	}
	source := "token"
	if strings.HasPrefix(token, "cookie:") {
		source = "cookie"
	}
	return Check{
		Name:    "auth_token",
		Status:  statusOK,
		Message: tr.T("output.doctor.auth_token.stored"),
		Details: map[string]interface{}{"source": source},
	}
}

func checkRepoContext(tr *i18n.Translator) Check {
	owner, repo, err := resolveOwnerRepo(cmdutil.Owner, cmdutil.Repo)
	if err != nil {
		return Check{
			Name:       "repo_context",
			Status:     statusWarning,
			Message:    tr.Tf("output.doctor.repo_context.missing", i18n.Args{"message": err.Error()}),
			Suggestion: tr.T("output.doctor.suggestion.pass_owner_repo"),
		}
	}
	return Check{
		Name:    "repo_context",
		Status:  statusOK,
		Message: tr.Tf("output.doctor.repo_context.ok", i18n.Args{"owner": owner, "repo": repo}),
		Details: map[string]interface{}{
			"owner": owner,
			"repo":  repo,
		},
	}
}

func checkAuthenticatedUser(tr *i18n.Translator, skipNetwork bool, cfgErr error) Check {
	if skipNetwork {
		return Check{
			Name:    "api_auth",
			Status:  statusSkipped,
			Message: tr.T("output.doctor.api_auth.skipped"),
		}
	}
	if cfgErr != nil {
		return Check{
			Name:       "api_auth",
			Status:     statusSkipped,
			Message:    tr.T("output.doctor.api_auth.config_skipped"),
			Suggestion: tr.T("output.doctor.suggestion.fix_config_yaml"),
		}
	}

	user, err := getCurrentUser()
	if err != nil {
		return Check{
			Name:       "api_auth",
			Status:     statusError,
			Message:    tr.Tf("output.doctor.api_auth.failed", i18n.Args{"message": err.Error()}),
			Suggestion: "gitlink-cli auth login",
		}
	}
	login, _ := user["login"].(string)
	if login == "" {
		return Check{
			Name:       "api_auth",
			Status:     statusWarning,
			Message:    tr.T("output.doctor.api_auth.no_login"),
			Suggestion: tr.T("output.doctor.suggestion.check_token"),
		}
	}
	return Check{
		Name:    "api_auth",
		Status:  statusOK,
		Message: tr.Tf("output.doctor.api_auth.ok", i18n.Args{"login": login}),
		Details: map[string]interface{}{
			"login": login,
		},
	}
}

func validateBaseURL(value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return err
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("scheme must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("host is required")
	}
	return nil
}

func validFormat(value string) bool {
	switch value {
	case "json", "table", "yaml":
		return true
	default:
		return false
	}
}

func resolveFormat() string {
	if cmdutil.Format != "" {
		return cmdutil.Format
	}
	return "json"
}

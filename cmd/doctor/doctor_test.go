package doctor

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/internal/i18n"
)

func TestDoctorSkipNetworkReportsLocalChecks(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: https://www.gitlink.org.cn/api\ndefault_format: json\n")
	t.Setenv("GITLINK_TOKEN", "secret-token")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "Gitlink", "gitlink-cli", nil
	}

	report := Run(true, i18n.Default())
	if !report.OK {
		t.Fatalf("expected report OK, got %+v", report)
	}
	assertCheck(t, report, "config_file", statusOK)
	assertCheck(t, report, "config_values", statusOK)
	assertCheck(t, report, "auth_token", statusOK)
	assertCheck(t, report, "repo_context", statusOK)
	assertCheck(t, report, "api_auth", statusSkipped)
	if report.Summary.Total != 5 {
		t.Fatalf("summary total = %d, want 5", report.Summary.Total)
	}
}

func TestDoctorInvalidConfigDoesNotPanic(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: [broken\n")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "Gitlink", "gitlink-cli", nil
	}

	report := Run(true, i18n.Default())
	if report.OK {
		t.Fatalf("expected report not OK, got %+v", report)
	}
	assertCheck(t, report, "config_file", statusError)
	assertCheck(t, report, "config_values", statusSkipped)
}

func TestDoctorInvalidBaseURL(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: gitlink.local/api\ndefault_format: table\n")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "Gitlink", "gitlink-cli", nil
	}

	report := Run(true, i18n.Default())
	if report.OK {
		t.Fatalf("expected invalid base_url to mark report not OK")
	}
	check := assertCheck(t, report, "config_values", statusError)
	if check.Suggestion == "" {
		t.Fatalf("expected config_values suggestion")
	}
}

func TestDoctorMissingRepoContextIsWarning(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: https://www.gitlink.org.cn/api\ndefault_format: table\n")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "", "", errors.New("no origin remote")
	}

	report := Run(true, i18n.Default())
	assertCheck(t, report, "auth_token", statusWarning)
	check := assertCheck(t, report, "repo_context", statusWarning)
	if check.Suggestion == "" {
		t.Fatalf("expected repo_context suggestion")
	}
	if !report.OK {
		t.Fatalf("warnings should not make report fail: %+v", report)
	}
}

func TestDoctorNetworkCheckCanSucceed(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: https://www.gitlink.org.cn/api\ndefault_format: table\n")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "Gitlink", "gitlink-cli", nil
	}
	getCurrentUser = func() (map[string]interface{}, error) {
		return map[string]interface{}{"login": "Mengz"}, nil
	}

	report := Run(false, i18n.Default())
	assertCheck(t, report, "api_auth", statusOK)
	if !report.OK {
		t.Fatalf("expected report OK, got %+v", report)
	}
}

func TestDoctorCommandPrintsJSONEnvelope(t *testing.T) {
	withDoctorTestState(t)
	writeConfig(t, "base_url: https://www.gitlink.org.cn/api\ndefault_format: json\n")
	resolveOwnerRepo = func(owner, repo string) (string, string, error) {
		return "Gitlink", "gitlink-cli", nil
	}

	cmd := NewDoctorCmd(i18n.Default())
	cmd.SetArgs([]string{"--skip-network"})
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var env struct {
		OK   bool            `json:"ok"`
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(out.Bytes(), &env); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, out.String())
	}
	if !env.OK || len(env.Data) == 0 {
		t.Fatalf("unexpected envelope: %+v", env)
	}
}

func withDoctorTestState(t *testing.T) {
	t.Helper()

	oldLoadConfig := loadConfig
	oldLoadToken := loadToken
	oldGetCurrentUser := getCurrentUser
	oldResolveOwnerRepo := resolveOwnerRepo
	oldStatFile := statFile
	oldLookupEnv := lookupEnv
	oldFormat := cmdutil.Format
	oldOwner := cmdutil.Owner
	oldRepo := cmdutil.Repo

	t.Setenv("GITLINK_CONFIG_DIR", t.TempDir())
	t.Setenv("GITLINK_TOKEN", "")
	cmdutil.Format = "json"
	cmdutil.Owner = ""
	cmdutil.Repo = ""
	loadConfig = oldLoadConfig
	loadToken = func() (string, error) { return "", os.ErrNotExist }
	getCurrentUser = func() (map[string]interface{}, error) {
		return nil, errors.New("unexpected network call")
	}
	resolveOwnerRepo = oldResolveOwnerRepo
	statFile = oldStatFile
	lookupEnv = func(key string) (string, bool) {
		if key == "GITLINK_TOKEN" {
			value := os.Getenv(key)
			return value, value != ""
		}
		return os.LookupEnv(key)
	}

	t.Cleanup(func() {
		loadConfig = oldLoadConfig
		loadToken = oldLoadToken
		getCurrentUser = oldGetCurrentUser
		resolveOwnerRepo = oldResolveOwnerRepo
		statFile = oldStatFile
		lookupEnv = oldLookupEnv
		cmdutil.Format = oldFormat
		cmdutil.Owner = oldOwner
		cmdutil.Repo = oldRepo
	})
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := os.Getenv("GITLINK_CONFIG_DIR")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
}

func assertCheck(t *testing.T, report Report, name, status string) Check {
	t.Helper()
	for _, check := range report.Checks {
		if check.Name == name {
			if check.Status != status {
				t.Fatalf("%s status = %s, want %s; check=%+v", name, check.Status, status, check)
			}
			return check
		}
	}
	t.Fatalf("missing check %q in %+v", name, report.Checks)
	return Check{}
}

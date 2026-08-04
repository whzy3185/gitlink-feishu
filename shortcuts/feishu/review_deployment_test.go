package feishu

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func reviewDeploymentFile(t *testing.T, parts ...string) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve deployment test file")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(current), "..", ".."))
	payload, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
	if err != nil {
		t.Fatal(err)
	}
	return string(payload)
}

func TestSystemdUnitDoesNotContainSecrets(t *testing.T) {
	unit := strings.ToLower(reviewDeploymentFile(t, "deploy", "systemd", "gitlink-feishu-review.service"))
	for _, forbidden := range []string{"--app-secret", "--admin-token", "--credential", "gitlink_review_token="} {
		if strings.Contains(unit, forbidden) {
			t.Fatalf("systemd unit contains secret argument %q", forbidden)
		}
	}
}

func TestSystemdUnitUsesSIGTERM(t *testing.T) {
	unit := reviewDeploymentFile(t, "deploy", "systemd", "gitlink-feishu-review.service")
	if !strings.Contains(unit, "KillSignal=SIGTERM") {
		t.Fatal("systemd unit does not use SIGTERM")
	}
}

func TestSystemdUnitHasStopTimeout(t *testing.T) {
	unit := reviewDeploymentFile(t, "deploy", "systemd", "gitlink-feishu-review.service")
	if !strings.Contains(unit, "TimeoutStopSec=45") {
		t.Fatal("systemd stop timeout missing")
	}
}

func TestSystemdUnitRestrictsWritablePaths(t *testing.T) {
	unit := reviewDeploymentFile(t, "deploy", "systemd", "gitlink-feishu-review.service")
	for _, required := range []string{"ProtectSystem=strict", "ProtectHome=true", "ReadWritePaths=/var/lib/gitlink-feishu-review /var/backups/gitlink-feishu-review"} {
		if !strings.Contains(unit, required) {
			t.Fatalf("systemd hardening missing %q", required)
		}
	}
}

func TestWindowsInstallSupportsWhatIf(t *testing.T) {
	script := reviewDeploymentFile(t, "deploy", "windows", "install-review-service.ps1")
	if !strings.Contains(script, "SupportsShouldProcess = $true") || !strings.Contains(script, "$PSCmdlet.ShouldProcess") {
		t.Fatal("Windows installer lacks WhatIf support")
	}
}

func TestWindowsUninstallPreservesDatabase(t *testing.T) {
	script := strings.ToLower(reviewDeploymentFile(t, "deploy", "windows", "uninstall-review-service.ps1"))
	if strings.Contains(script, "remove-item") || !strings.Contains(script, "preserv") {
		t.Fatal("Windows uninstall does not explicitly preserve state")
	}
}

func TestWindowsScriptsDoNotPutSecretsInArguments(t *testing.T) {
	for _, name := range []string{"install-review-service.ps1", "uninstall-review-service.ps1", "start-review-service.ps1", "stop-review-service.ps1", "restart-review-service.ps1"} {
		script := strings.ToLower(reviewDeploymentFile(t, "deploy", "windows", name))
		for _, forbidden := range []string{"--app-secret", "--admin-token", "--credential-ref", "--webhook-secret"} {
			if strings.Contains(script, forbidden) {
				t.Fatalf("%s puts secret in arguments: %s", name, forbidden)
			}
		}
	}
}

func TestReverseProxyDoesNotExposeAdminAPI(t *testing.T) {
	for _, path := range [][]string{{"deploy", "caddy", "Caddyfile.example"}, {"deploy", "nginx", "gitlink-feishu-review.conf.example"}} {
		config := strings.ToLower(reviewDeploymentFile(t, path...))
		if strings.Contains(config, "/admin/") || strings.Contains(config, "/metrics") {
			t.Fatalf("proxy %v exposes local administration", path)
		}
	}
}

func TestReverseProxyPreservesWebhookHeaders(t *testing.T) {
	for _, path := range [][]string{{"deploy", "caddy", "Caddyfile.example"}, {"deploy", "nginx", "gitlink-feishu-review.conf.example"}} {
		config := strings.ToLower(reviewDeploymentFile(t, path...))
		for _, header := range []string{"x-gitlink-signature", "x-gitlink-timestamp", "x-gitlink-delivery"} {
			if !strings.Contains(config, header) {
				t.Fatalf("proxy %v misses %s", path, header)
			}
		}
	}
}

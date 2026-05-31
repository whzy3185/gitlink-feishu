package context

import (
	"testing"
)

func TestParseRemoteURLHTTPS(t *testing.T) {
	owner, repo, err := parseRemoteURL("https://www.gitlink.org.cn/Gitlink/gitlink-cli.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "Gitlink" {
		t.Fatalf("owner = %q, want Gitlink", owner)
	}
	if repo != "gitlink-cli" {
		t.Fatalf("repo = %q, want gitlink-cli", repo)
	}
}

func TestParseRemoteURLHTTPSNoGit(t *testing.T) {
	owner, repo, err := parseRemoteURL("https://www.gitlink.org.cn/owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "owner" || repo != "repo" {
		t.Fatalf("got %s/%s, want owner/repo", owner, repo)
	}
}

func TestParseRemoteURLSSH(t *testing.T) {
	owner, repo, err := parseRemoteURL("git@www.gitlink.org.cn:Gitlink/gitlink-cli.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "Gitlink" {
		t.Fatalf("owner = %q, want Gitlink", owner)
	}
	if repo != "gitlink-cli" {
		t.Fatalf("repo = %q, want gitlink-cli", repo)
	}
}

func TestParseRemoteURLSSHNoSuffix(t *testing.T) {
	owner, repo, err := parseRemoteURL("git@gitlink.org.cn:owner/repo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "owner" || repo != "repo" {
		t.Fatalf("got %s/%s, want owner/repo", owner, repo)
	}
}

func TestParseRemoteURLInvalidSSH(t *testing.T) {
	_, _, err := parseRemoteURL("git@gitlink.org.cn")
	if err == nil {
		t.Fatal("expected error for invalid SSH URL")
	}
}

func TestParseRemoteURLInvalidHTTPS(t *testing.T) {
	_, _, err := parseRemoteURL("://invalid-url")
	if err == nil {
		t.Fatal("expected error for invalid HTTPS URL")
	}
}

func TestParsePathSegments(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{"basic", "owner/repo", "owner", "repo", false},
		{"with git", "owner/repo.git", "owner", "repo", false},
		{"leading slash", "/owner/repo", "owner", "repo", false},
		{"both", "/owner/repo.git", "owner", "repo", false},
		{"with subpath", "owner/repo/sub", "owner", "repo", false},
		{"single segment", "onlyowner", "", "", true},
		{"empty", "", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parsePathSegments(tt.path)
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Fatalf("got %s/%s, want %s/%s", owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}

func TestResolveOwnerRepoExplicit(t *testing.T) {
	owner, repo, err := ResolveOwnerRepo("explicitOwner", "explicitRepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "explicitOwner" || repo != "explicitRepo" {
		t.Fatalf("got %s/%s, want explicitOwner/explicitRepo", owner, repo)
	}
}

func TestResolveOwnerRepoPartialFlagsInGitRepo(t *testing.T) {
	// When in a git repo, partial flags use git remote for the missing part.
	owner, repo, err := ResolveOwnerRepo("", "partialRepo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner == "" {
		t.Fatal("expected owner to be resolved from git remote")
	}
	if repo != "partialRepo" {
		t.Fatalf("repo = %q, want partialRepo", repo)
	}

	owner, repo, err = ResolveOwnerRepo("partialOwner", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "partialOwner" {
		t.Fatalf("owner = %q, want partialOwner", owner)
	}
	if repo == "" {
		t.Fatal("expected repo to be resolved from git remote")
	}
}

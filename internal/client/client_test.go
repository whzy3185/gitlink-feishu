package client

import "testing"

func TestNormalizeAPIPathStripsDuplicateAPIPrefix(t *testing.T) {
	got := normalizeAPIPath("https://www.gitlink.org.cn/api", "/api/v1/repos/Gitlink/gitlink-cli/contents/README.md")
	want := "/v1/repos/Gitlink/gitlink-cli/contents/README.md"
	if got != want {
		t.Fatalf("normalizeAPIPath() = %q, want %q", got, want)
	}
}

func TestNormalizeAPIPathKeepsRegularPath(t *testing.T) {
	got := normalizeAPIPath("https://www.gitlink.org.cn/api", "/projects")
	want := "/projects"
	if got != want {
		t.Fatalf("normalizeAPIPath() = %q, want %q", got, want)
	}
}

func TestNormalizeAPIPathKeepsAPIPrefixForNonAPIBaseURL(t *testing.T) {
	got := normalizeAPIPath("https://www.gitlink.org.cn", "/api/v1/repos/Gitlink/gitlink-cli")
	want := "/api/v1/repos/Gitlink/gitlink-cli"
	if got != want {
		t.Fatalf("normalizeAPIPath() = %q, want %q", got, want)
	}
}

package browse

import "testing"

func TestWebRoot(t *testing.T) {
	cases := map[string]string{
		"https://www.gitlink.org.cn/api":  "https://www.gitlink.org.cn",
		"https://www.gitlink.org.cn/api/": "https://www.gitlink.org.cn",
		"http://localhost:3000/api":       "http://localhost:3000",
		"https://example.com":             "https://example.com",
	}
	for in, want := range cases {
		if got := webRoot(in); got != want {
			t.Errorf("webRoot(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoRefURL(t *testing.T) {
	r := repoRef{host: "https://www.gitlink.org.cn", owner: "Gitlink", repo: "gitlink-cli"}
	cases := []struct {
		name string
		got  string
		want string
	}{
		{"home", r.url(), "https://www.gitlink.org.cn/Gitlink/gitlink-cli"},
		{"issue", r.url("issues", "42"), "https://www.gitlink.org.cn/Gitlink/gitlink-cli/issues/42"},
		{"pr", r.url("pulls", "262"), "https://www.gitlink.org.cn/Gitlink/gitlink-cli/pulls/262"},
		{"commit", r.url("commits", "abc123"), "https://www.gitlink.org.cn/Gitlink/gitlink-cli/commits/abc123"},
		{"branch tree", r.url("tree", "master"), "https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master"},
		{"releases", r.url("releases"), "https://www.gitlink.org.cn/Gitlink/gitlink-cli/releases"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, c.got, c.want)
		}
	}
}

func TestRepoRefURLMultiSegmentAndEscaping(t *testing.T) {
	r := repoRef{host: "https://www.gitlink.org.cn", owner: "Gitlink", repo: "gitlink-cli"}

	// A file path keeps its slashes across segments.
	if got, want := r.url("tree", "master", "skills/README.md"),
		"https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/skills/README.md"; got != want {
		t.Errorf("file path: got %q, want %q", got, want)
	}

	// A branch name with a slash is preserved, not collapsed.
	if got, want := r.url("tree", "feat/browse"),
		"https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/feat/browse"; got != want {
		t.Errorf("slash branch: got %q, want %q", got, want)
	}

	// A path segment with a space is percent-escaped.
	if got, want := r.url("tree", "master", "my dir/file.txt"),
		"https://www.gitlink.org.cn/Gitlink/gitlink-cli/tree/master/my%20dir/file.txt"; got != want {
		t.Errorf("escaping: got %q, want %q", got, want)
	}
}

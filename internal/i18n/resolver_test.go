package i18n

import "testing"

func TestPreScanLang(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"--lang", "zh-CN", "repo"}, "zh-CN"},
		{[]string{"repo", "--lang=zh-CN"}, "zh-CN"},
		{[]string{"repo"}, ""},
	}
	for _, tc := range cases {
		if got := PreScanLang(tc.args); got != tc.want {
			t.Fatalf("PreScanLang(%v) = %q, want %q", tc.args, got, tc.want)
		}
	}
}

func TestResolveLocalePriority(t *testing.T) {
	available := []string{"en-US", "zh-CN"}
	got := ResolveLocale(ResolveOptions{
		ExplicitLang: "en-US",
		Env:          map[string]string{"GITLINK_LANG": "zh-CN"},
		ConfigLang:   "zh-CN",
	}, available)
	if got != "en-US" {
		t.Fatalf("ResolveLocale() = %q, want en-US", got)
	}
}

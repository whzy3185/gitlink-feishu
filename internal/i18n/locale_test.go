package i18n

import "testing"

func TestNormalizeLocale(t *testing.T) {
	cases := map[string]string{
		"zh_CN":          "zh-CN",
		"zh_CN.UTF-8":    "zh-CN",
		"zh-Hans-CN":     "zh-Hans-CN",
		"EN_us":          "en-US",
		"  en-US  ":      "en-US",
		"zh-hans-cn.utf": "zh-Hans-CN",
	}
	for input, want := range cases {
		if got := NormalizeLocale(input); got != want {
			t.Fatalf("NormalizeLocale(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMatchLocale(t *testing.T) {
	available := []string{"en-US", "zh-CN"}
	cases := map[string]string{
		"zh_CN":      "zh-CN",
		"zh-Hans-CN": "zh-CN",
		"zh":         "zh-CN",
		"en":         "en-US",
		"fr-FR":      "en-US",
	}
	for input, want := range cases {
		if got := MatchLocale(input, available, "en-US"); got != want {
			t.Fatalf("MatchLocale(%q) = %q, want %q", input, got, want)
		}
	}
}

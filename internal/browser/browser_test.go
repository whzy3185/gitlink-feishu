package browser

import (
	"reflect"
	"testing"
)

func TestCommandPlatformDefaults(t *testing.T) {
	cases := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{"windows", "rundll32", []string{"url.dll,FileProtocolHandler", "https://x"}},
		{"darwin", "open", []string{"https://x"}},
		{"linux", "xdg-open", []string{"https://x"}},
		{"freebsd", "xdg-open", []string{"https://x"}},
	}
	for _, c := range cases {
		name, args := Command(c.goos, "", "https://x")
		if name != c.wantName || !reflect.DeepEqual(args, c.wantArgs) {
			t.Errorf("Command(%q) = %q %v, want %q %v", c.goos, name, args, c.wantName, c.wantArgs)
		}
	}
}

func TestCommandBrowserEnvOverridesPlatform(t *testing.T) {
	name, args := Command("linux", "firefox", "https://x")
	if name != "firefox" || !reflect.DeepEqual(args, []string{"https://x"}) {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestCommandBrowserEnvWithFlags(t *testing.T) {
	name, args := Command("windows", "chrome --incognito", "https://x")
	if name != "chrome" || !reflect.DeepEqual(args, []string{"--incognito", "https://x"}) {
		t.Fatalf("got %q %v", name, args)
	}
}

func TestCommandBlankBrowserEnvFallsBack(t *testing.T) {
	name, _ := Command("darwin", "   ", "https://x")
	if name != "open" {
		t.Fatalf("blank BROWSER should fall back to platform default, got %q", name)
	}
}

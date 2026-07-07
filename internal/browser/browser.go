// Package browser opens URLs in the user's default web browser.
package browser

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// Command resolves the executable and arguments used to open url on goos.
// A non-empty browserEnv (the $BROWSER value) overrides the platform default,
// so users on headless or non-standard setups can point at their own launcher.
func Command(goos, browserEnv, url string) (string, []string) {
	if fields := strings.Fields(browserEnv); len(fields) > 0 {
		return fields[0], append(fields[1:], url)
	}
	switch goos {
	case "windows":
		return "rundll32", []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		return "open", []string{url}
	default:
		return "xdg-open", []string{url}
	}
}

// Open launches the default browser pointed at url without blocking on it.
func Open(url string) error {
	name, args := Command(runtime.GOOS, os.Getenv("BROWSER"), url)
	return exec.Command(name, args...).Start()
}

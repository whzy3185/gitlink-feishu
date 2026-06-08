package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed index.html
var indexHTML []byte

type RunResult struct {
	OK      bool        `json:"ok"`
	Command string      `json:"command"`
	Output  interface{} `json:"output"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9090"
	}
	cliBin := findCLIBinary()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})

	http.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		module := r.URL.Query().Get("module")
		command := r.URL.Query().Get("command")
		owner := r.URL.Query().Get("owner")
		repo := r.URL.Query().Get("repo")
		format := r.URL.Query().Get("format")
		extraArgs := r.URL.Query().Get("args")

		if module == "" || command == "" {
			json.NewEncoder(w).Encode(RunResult{Error: "missing module or command"})
			return
		}
		if owner == "" {
			owner = "chroe"
		}
		if repo == "" {
			if module == "wiki" {
				repo = "gitlink_help_center"
			} else {
				repo = "gitlink-cli"
			}
		}
		if format == "" {
			format = "json"
		}

		args := []string{module, "+" + command, "--owner", owner, "--repo", repo, "--format", format}
		if extraArgs != "" {
			args = append(args, parseShellArgs(extraArgs)...)
		}

		cmdStr := "gitlink-cli " + strings.Join(args, " ")
		log.Printf("Running: %s", cmdStr)

		cmd := exec.Command(cliBin, args...)
		output, err := cmd.CombinedOutput()

		result := RunResult{
			Command: cmdStr,
		}

		if err != nil {
			result.Error = strings.TrimSpace(string(output))
			result.Output = nil
		} else {
			result.OK = true
			var parsed interface{}
			if json.Unmarshal(output, &parsed) == nil {
				result.Output = parsed
			} else {
				result.Output = strings.TrimSpace(string(output))
			}
		}

		json.NewEncoder(w).Encode(result)
	})

	fmt.Printf("Showcase Dashboard running at http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// parseShellArgs splits a shell-style argument string, respecting quoted values.
// e.g. `--content "Hello Wiki!" --message "create page"` -> ["--content", "Hello Wiki!", "--message", "create page"]
func parseShellArgs(s string) []string {
	var args []string
	var current strings.Builder
	inQuote := false

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' {
			inQuote = !inQuote
			continue
		}
		if ch == ' ' && !inQuote {
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteByte(ch)
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func findCLIBinary() string {
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	candidates := []string{
		filepath.Join(exeDir, "gitlink-cli.exe"),
		filepath.Join(exeDir, "gitlink-cli"),
		filepath.Join(exeDir, "..", "gitlink-cli.exe"),
		filepath.Join(exeDir, "..", "gitlink-cli"),
		"./gitlink-cli.exe",
		"./gitlink-cli",
		"../gitlink-cli.exe",
		"../gitlink-cli",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return "gitlink-cli"
}

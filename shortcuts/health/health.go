package health

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	return []*common.Shortcut{
		{
			Name:        "fetch",
			Description: "Fetch PR and Issue data into SQLite for health analysis",
			Flags: []common.Flag{
				{Name: "db", Short: "d", Usage: "SQLite database path (default: ~/.agents/skills/gitlink-health/data/gitlink_health.db)"},
				{Name: "max-pages", Short: "M", Usage: "Maximum pages per query (default: unlimited)"},
			},
			Run: func(ctx *common.RuntimeContext) error {
				if err := ctx.ResolveOwnerRepo(); err != nil {
					return err
				}

				// maxPages: 0 means unlimited
				maxPages := 0
				if v := ctx.Arg("max-pages"); v != "" {
					if n, err := strconv.Atoi(v); err == nil && n > 0 {
						maxPages = n
					}
				}

				dbPath := ctx.Arg("db")
				if dbPath == "" {
					home, _ := os.UserHomeDir()
					dbPath = filepath.Join(home, ".agents", "skills", "gitlink-health", "data", "gitlink_health.db")
				}

				fmt.Fprintf(os.Stderr, "\nFetching data for %s/%s...\n", ctx.Owner, ctx.Repo)

				db, err := openDB(dbPath)
				if err != nil {
					return err
				}
				defer db.Close()

				repoID, err := getOrCreateRepo(db, ctx.Repo, ctx.Owner)
				if err != nil {
					return fmt.Errorf("create repo: %w", err)
				}

				// ── Fetch PRs ──
				fmt.Fprintf(os.Stderr, "\n=== Fetching Pull Requests ===\n")
				seenPRIDs := make(map[int]bool)
				var prAgg map[string]interface{}

				for _, state := range []string{"open", "closed", "merged"} {
					page := 1
					for maxPages == 0 || page <= maxPages {
						fmt.Fprintf(os.Stderr, "  PR list: state=%s, page=%d...\n", state, page)
						prs, agg := fetchPRListPage(ctx, state, page, 20)
						if agg != nil {
							prAgg = agg
						}
						if len(prs) == 0 {
							break
						}
						for _, item := range prs {
							pr, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							var prID int
							if v, ok := pr["pull_request_id"].(float64); ok && v > 0 {
								prID = int(v)
							} else if v, ok := pr["id"].(float64); ok {
								prID = int(v)
							} else {
								continue
							}
							if seenPRIDs[prID] {
								continue
							}
							seenPRIDs[prID] = true
							savePull(db, repoID, pr)
						}
						if len(prs) < 20 {
							break
						}
						page++
						sleep()
					}
				}

				fmt.Fprintf(os.Stderr, "  Total PRs: %d\n", len(seenPRIDs))
				if prAgg != nil {
					fmt.Fprintf(os.Stderr, "  API aggregates: total=%v, merged=%v, open=%v, closed=%v\n",
						prAgg["search_count"], prAgg["merged_issues_size"],
						prAgg["open_count"], prAgg["close_count"])
				}

				// ── Fetch Issues ──
				fmt.Fprintf(os.Stderr, "\n=== Fetching Issues ===\n")
				seenIssueIDs := make(map[int]bool)
				var issueAgg map[string]interface{}

				for _, state := range []string{"open", "closed"} {
					page := 1
					for maxPages == 0 || page <= maxPages {
						fmt.Fprintf(os.Stderr, "  Issue list: state=%s, page=%d...\n", state, page)
						issues, agg := fetchIssueListPage(ctx, ctx.Owner, ctx.Repo, state, page, 20)
						if agg != nil {
							issueAgg = agg
						}
						if len(issues) == 0 {
							break
						}
						for _, item := range issues {
							issue, ok := item.(map[string]interface{})
							if !ok {
								continue
							}
							issueID, ok := issue["id"].(float64)
							if !ok || seenIssueIDs[int(issueID)] {
								continue
							}
							seenIssueIDs[int(issueID)] = true

							// Get issue number (project_issues_index) from list item
							issueNumber := int(issueID)
							if v, ok := issue["project_issues_index"].(float64); ok && v > 0 {
								issueNumber = int(v)
							}

							listUpdatedAt, _ := issue["updated_at"].(string)
							saveIssue(db, repoID, issue, issueNumber, listUpdatedAt)
						}
						if len(issues) < 20 {
							break
						}
						page++
						sleep()
					}
				}

				fmt.Fprintf(os.Stderr, "  Total Issues: %d\n", len(seenIssueIDs))
				if issueAgg != nil {
					fmt.Fprintf(os.Stderr, "  API aggregates: total=%v, open=%v, closed=%v\n",
						issueAgg["total_count"], issueAgg["opened_count"], issueAgg["closed_count"])
				}

				fmt.Fprintf(os.Stderr, "\nData saved to %s\n", dbPath)
				return nil
			},
		},
	}
}

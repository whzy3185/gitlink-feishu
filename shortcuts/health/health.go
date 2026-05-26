package health

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
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

				limiter := rate.NewLimiter(rate.Limit(1.0/0.6), 5)

				s := &sharedState{
					seenPRs:    make(map[int]bool),
					seenIssues: make(map[int]bool),
				}

				fmt.Fprintf(os.Stderr, "\n=== Fetching Pull Requests ===\n")
				fmt.Fprintf(os.Stderr, "=== Fetching Issues ===\n")

				g, egCtx := errgroup.WithContext(context.Background())

				for _, state := range []string{"open", "closed", "merged"} {
					state := state
					g.Go(func() error {
						return fetchPRs(ctx, db, repoID, state, maxPages, limiter, egCtx, s)
					})
				}
				for _, state := range []string{"open", "closed"} {
					state := state
					g.Go(func() error {
						return fetchIssues(ctx, db, repoID, state, maxPages, limiter, egCtx, s)
					})
				}

				if err := g.Wait(); err != nil {
					return err
				}

				fmt.Fprintf(os.Stderr, "  Total PRs: %d\n", s.prCount)

				fmt.Fprintf(os.Stderr, "  Total Issues: %d\n", s.issueCount)

				fmt.Fprintf(os.Stderr, "\nData saved to %s\n", dbPath)
				return nil
			},
		},
	}
}

type sharedState struct {
	mu         sync.Mutex
	seenPRs    map[int]bool
	seenIssues map[int]bool
	prCount    int
	issueCount int
}

func fetchPRs(ctx *common.RuntimeContext, db *sql.DB, repoID int, state string, maxPages int, limiter *rate.Limiter, egCtx context.Context, s *sharedState) error {
	page := 1
	for maxPages == 0 || page <= maxPages {
		if egCtx.Err() != nil {
			return nil
		}
		if err := limiter.Wait(egCtx); err != nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "  PR list: state=%s, page=%d...\n", state, page)
		prs, err := fetchPRListPage(ctx, state, page, 20)
		if err != nil {
			return err
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
			s.mu.Lock()
			dup := s.seenPRs[prID]
			if !dup {
				s.seenPRs[prID] = true
				s.prCount++
			}
			s.mu.Unlock()
			if dup {
				continue
			}

			// For merged PRs: try list response first, fall back to detail API
			mergedAt := ""
			if state == "merged" {
				mergedAt = mergeTimeFromList(pr)
				if mergedAt == "" {
					if prNumFloat, ok := pr["pull_request_number"].(float64); ok && prNumFloat > 0 {
						prNum := int(prNumFloat)
						if err := limiter.Wait(egCtx); err != nil {
							return nil
						}
						fmt.Fprintf(os.Stderr, "  PR detail #%d...\n", prNum)
						if ma, err := fetchPRDetail(ctx, prNum); err != nil {
							fmt.Fprintf(os.Stderr, "  PR detail #%d: %v\n", prNum, err)
						} else {
							mergedAt = ma
						}
					}
				}
			}

			savePull(db, repoID, pr, mergedAt)
		}
		if len(prs) < 20 {
			break
		}
		page++
	}
	return nil
}

func fetchIssues(ctx *common.RuntimeContext, db *sql.DB, repoID int, state string, maxPages int, limiter *rate.Limiter, egCtx context.Context, s *sharedState) error {
	page := 1
	for maxPages == 0 || page <= maxPages {
		if egCtx.Err() != nil {
			return nil
		}
		if err := limiter.Wait(egCtx); err != nil {
			return nil
		}
		fmt.Fprintf(os.Stderr, "  Issue list: state=%s, page=%d...\n", state, page)
		issues, err := fetchIssueListPage(ctx, ctx.Owner, ctx.Repo, state, page, 20)
		if err != nil {
			return err
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
			if !ok {
				continue
			}
			s.mu.Lock()
			dup := s.seenIssues[int(issueID)]
			if !dup {
				s.seenIssues[int(issueID)] = true
				s.issueCount++
			}
			s.mu.Unlock()
			if dup {
				continue
			}
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
	}
	return nil
}

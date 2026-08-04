package compare

import (
	"fmt"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/output"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	defaultCompareCommitsLimit       = 20
	defaultCompareSummaryCommitLimit = 10
	defaultCompareSummaryTopFiles    = 10
	defaultCompareSummaryMaxFiles    = 200
)

type compareCommit struct {
	SHA            string `json:"sha"`
	Subject        string `json:"subject"`
	Message        string `json:"message,omitempty"`
	CreatedAt      string `json:"created_at,omitempty"`
	TimeFromNow    string `json:"time_from_now,omitempty"`
	AuthorLogin    string `json:"author_login,omitempty"`
	AuthorName     string `json:"author_name,omitempty"`
	CommitterLogin string `json:"committer_login,omitempty"`
	CommitterName  string `json:"committer_name,omitempty"`
}

type compareCommitsResult struct {
	Repository      string          `json:"repository"`
	Head            string          `json:"head"`
	Base            string          `json:"base"`
	CompareMessage  string          `json:"compare_message,omitempty"`
	TotalCommits    int             `json:"total_commits"`
	MatchedCommits  int             `json:"matched_commits"`
	ReturnedCommits int             `json:"returned_commits"`
	Truncated       bool            `json:"truncated"`
	AuthorFilter    string          `json:"author_filter,omitempty"`
	Keyword         string          `json:"keyword,omitempty"`
	Reversed        bool            `json:"reversed"`
	Commits         []compareCommit `json:"commits"`
}

type compareFile struct {
	Filename    string
	OldName     string
	Additions   int
	Deletions   int
	Changes     int
	IsCreated   bool
	IsDeleted   bool
	IsRenamed   bool
	IsBinary    bool
	IsSubmodule bool
}

type compareFilesPage struct {
	Files          []compareFile
	TotalFiles     int
	TotalAdditions int
	TotalDeletions int
}

type compareFilesFetchResult struct {
	Files          []compareFile
	TotalFiles     int
	TotalAdditions int
	TotalDeletions int
	Truncated      bool
}

type compareChangeTotals struct {
	Additions int `json:"additions"`
	Deletions int `json:"deletions"`
	Changes   int `json:"changes"`
}

type compareFileTypeSummary struct {
	Created   int `json:"created"`
	Modified  int `json:"modified"`
	Deleted   int `json:"deleted"`
	Renamed   int `json:"renamed"`
	Binary    int `json:"binary"`
	Submodule int `json:"submodule"`
}

type compareBucket struct {
	Name      string `json:"name"`
	Files     int    `json:"files"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
}

type compareFileSummary struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Changes   int    `json:"changes"`
}

type compareSummaryResult struct {
	Repository     string                 `json:"repository"`
	Head           string                 `json:"head"`
	Base           string                 `json:"base"`
	CompareMessage string                 `json:"compare_message,omitempty"`
	CommitsCount   int                    `json:"commits_count"`
	FilesCount     int                    `json:"files_count"`
	FilesAnalyzed  int                    `json:"files_analyzed"`
	TruncatedFiles bool                   `json:"truncated_files"`
	Authors        []string               `json:"authors"`
	CommitsSample  []compareCommit        `json:"commits_sample"`
	ChangeTotals   compareChangeTotals    `json:"change_totals"`
	FileTypes      compareFileTypeSummary `json:"file_types"`
	PathGroups     []compareBucket        `json:"path_groups"`
	Extensions     []compareBucket        `json:"extensions"`
	TopFiles       []compareFileSummary   `json:"top_files"`
}

func resolveCompareRefs(ctx *common.RuntimeContext) (string, string, error) {
	head, err := ctx.RequireArg("head")
	if err != nil {
		return "", "", err
	}
	base, err := ctx.RequireArg("base")
	if err != nil {
		return "", "", err
	}
	return head, base, nil
}

func runCompareCommits(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	head, base, err := resolveCompareRefs(ctx)
	if err != nil {
		return err
	}
	limit, err := parsePositiveIntArg(ctx.Arg("limit"), defaultCompareCommitsLimit, "limit")
	if err != nil {
		return err
	}
	viewData, err := fetchCompareViewData(ctx, head, base)
	if err != nil {
		return err
	}
	result := buildCompareCommitsResult(
		fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		head,
		base,
		viewData,
		ctx.Arg("author"),
		ctx.Arg("keyword"),
		limit,
		ctx.Arg("reverse") == "true",
	)
	return ctx.OutputData(result)
}

func runCompareSummary(ctx *common.RuntimeContext) error {
	if err := ctx.ResolveOwnerRepo(); err != nil {
		return err
	}
	head, base, err := resolveCompareRefs(ctx)
	if err != nil {
		return err
	}
	maxFiles, err := parsePositiveIntArg(ctx.Arg("max-files"), defaultCompareSummaryMaxFiles, "max-files")
	if err != nil {
		return err
	}
	topFiles, err := parsePositiveIntArg(ctx.Arg("top-files"), defaultCompareSummaryTopFiles, "top-files")
	if err != nil {
		return err
	}
	commitLimit, err := parsePositiveIntArg(ctx.Arg("commit-limit"), defaultCompareSummaryCommitLimit, "commit-limit")
	if err != nil {
		return err
	}
	viewData, err := fetchCompareViewData(ctx, head, base)
	if err != nil {
		return err
	}
	filesResult, err := fetchCompareFilesForSummary(ctx, head, base, maxFiles)
	if err != nil {
		return err
	}
	result := buildCompareSummary(
		fmt.Sprintf("%s/%s", ctx.Owner, ctx.Repo),
		head,
		base,
		viewData,
		filesResult,
		commitLimit,
		topFiles,
	)
	return ctx.OutputData(result)
}

func parsePositiveIntArg(raw string, defaultValue int, flagName string) (int, error) {
	if strings.TrimSpace(raw) == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, fmt.Errorf("invalid --%s value %q: use a positive integer", flagName, raw)
	}
	return value, nil
}

func fetchCompareViewData(ctx *common.RuntimeContext, head, base string) (map[string]interface{}, error) {
	env, err := ctx.CallAPI("GET", comparePath(ctx, head, base), nil)
	if err != nil {
		return nil, err
	}
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected compare response format")
	}
	return data, nil
}

func fetchCompareFilesForSummary(ctx *common.RuntimeContext, head, base string, maxFiles int) (compareFilesFetchResult, error) {
	result := compareFilesFetchResult{}
	pageSize := 100
	if maxFiles < pageSize {
		pageSize = maxFiles
	}
	for page := 1; len(result.Files) < maxFiles; page++ {
		remaining := maxFiles - len(result.Files)
		limit := pageSize
		if remaining < limit {
			limit = remaining
		}
		pageResult, err := fetchCompareFilesPage(ctx, head, base, page, limit)
		if err != nil {
			return compareFilesFetchResult{}, err
		}
		if page == 1 {
			result.TotalFiles = pageResult.TotalFiles
			result.TotalAdditions = pageResult.TotalAdditions
			result.TotalDeletions = pageResult.TotalDeletions
		}
		result.Files = append(result.Files, pageResult.Files...)
		if len(pageResult.Files) < limit || len(result.Files) >= result.TotalFiles {
			break
		}
	}
	if result.TotalFiles == 0 {
		result.TotalFiles = len(result.Files)
	}
	result.Truncated = result.TotalFiles > len(result.Files)
	return result, nil
}

func fetchCompareFilesPage(ctx *common.RuntimeContext, head, base string, page, limit int) (compareFilesPage, error) {
	q := url.Values{}
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(limit))
	env, err := ctx.CallAPIWithQuery("GET", "/v1"+comparePath(ctx, head, base)+"/files", q)
	if err != nil {
		return compareFilesPage{}, err
	}
	return parseCompareFilesPage(env)
}

func parseCompareFilesPage(env *output.Envelope) (compareFilesPage, error) {
	data, ok := env.Data.(map[string]interface{})
	if !ok {
		return compareFilesPage{}, fmt.Errorf("unexpected compare files response format")
	}
	rawFiles, ok := data["files"].([]interface{})
	if !ok {
		return compareFilesPage{}, fmt.Errorf("compare files response missing files list")
	}
	files := make([]compareFile, 0, len(rawFiles))
	for _, raw := range rawFiles {
		file, ok := normalizeCompareFile(raw)
		if ok {
			files = append(files, file)
		}
	}
	totalFiles := intField(data, "file_nums", "files_count")
	if totalFiles == 0 {
		totalFiles = len(files)
	}
	return compareFilesPage{
		Files:          files,
		TotalFiles:     totalFiles,
		TotalAdditions: intField(data, "total_addition"),
		TotalDeletions: intField(data, "total_deletion"),
	}, nil
}

func buildCompareCommitsResult(repository, head, base string, viewData map[string]interface{}, authorFilter, keyword string, limit int, reverse bool) compareCommitsResult {
	commits := extractCompareCommits(viewData)
	totalCommits := intField(viewData, "commits_count")
	if totalCommits == 0 {
		totalCommits = len(commits)
	}
	filtered := filterCompareCommits(commits, authorFilter, keyword)
	if reverse {
		reverseCompareCommits(filtered)
	}
	matchedCommits := len(filtered)
	truncated := false
	if len(filtered) > limit {
		filtered = filtered[:limit]
		truncated = true
	}
	return compareCommitsResult{
		Repository:      repository,
		Head:            head,
		Base:            base,
		CompareMessage:  stringField(viewData, "message"),
		TotalCommits:    totalCommits,
		MatchedCommits:  matchedCommits,
		ReturnedCommits: len(filtered),
		Truncated:       truncated,
		AuthorFilter:    strings.TrimSpace(authorFilter),
		Keyword:         strings.TrimSpace(keyword),
		Reversed:        reverse,
		Commits:         filtered,
	}
}

func buildCompareSummary(repository, head, base string, viewData map[string]interface{}, filesResult compareFilesFetchResult, commitLimit, topFiles int) compareSummaryResult {
	commits := extractCompareCommits(viewData)
	commitsCount := intField(viewData, "commits_count")
	if commitsCount == 0 {
		commitsCount = len(commits)
	}
	if len(commits) > commitLimit {
		commits = commits[:commitLimit]
	}
	filesCount := filesResult.TotalFiles
	if filesCount == 0 {
		filesCount = intField(viewData, "files_count")
	}
	if filesCount == 0 {
		filesCount = len(filesResult.Files)
	}
	totalAdditions, totalDeletions := compareTotalsFromFiles(filesResult)
	return compareSummaryResult{
		Repository:     repository,
		Head:           head,
		Base:           base,
		CompareMessage: stringField(viewData, "message"),
		CommitsCount:   commitsCount,
		FilesCount:     filesCount,
		FilesAnalyzed:  len(filesResult.Files),
		TruncatedFiles: filesResult.Truncated,
		Authors:        uniqueCompareAuthors(extractCompareCommits(viewData)),
		CommitsSample:  commits,
		ChangeTotals: compareChangeTotals{
			Additions: totalAdditions,
			Deletions: totalDeletions,
			Changes:   totalAdditions + totalDeletions,
		},
		FileTypes:  summarizeCompareFileTypes(filesResult.Files),
		PathGroups: summarizeCompareBuckets(filesResult.Files, pathGroupForFile, 10),
		Extensions: summarizeCompareBuckets(filesResult.Files, extensionForFile, 10),
		TopFiles:   topCompareFiles(filesResult.Files, topFiles),
	}
}

func extractCompareCommits(data map[string]interface{}) []compareCommit {
	rawCommits, ok := data["commits"].([]interface{})
	if !ok {
		return nil
	}
	commits := make([]compareCommit, 0, len(rawCommits))
	for _, raw := range rawCommits {
		commit, ok := normalizeCompareCommit(raw)
		if ok {
			commits = append(commits, commit)
		}
	}
	return commits
}

func normalizeCompareCommit(raw interface{}) (compareCommit, bool) {
	data, ok := raw.(map[string]interface{})
	if !ok {
		return compareCommit{}, false
	}
	message := strings.TrimSpace(stringField(data, "message"))
	author := nestedMap(data, "author")
	committer := nestedMap(data, "committer")
	return compareCommit{
		SHA:            stringField(data, "sha"),
		Subject:        subjectFromMessage(message),
		Message:        message,
		CreatedAt:      stringField(data, "created_at"),
		TimeFromNow:    stringField(data, "time_from_now"),
		AuthorLogin:    stringField(author, "login"),
		AuthorName:     stringField(author, "name"),
		CommitterLogin: stringField(committer, "login"),
		CommitterName:  stringField(committer, "name"),
	}, true
}

func normalizeCompareFile(raw interface{}) (compareFile, bool) {
	data, ok := raw.(map[string]interface{})
	if !ok {
		return compareFile{}, false
	}
	return compareFile{
		Filename:    stringField(data, "filename"),
		OldName:     stringField(data, "old_name"),
		Additions:   intField(data, "additions"),
		Deletions:   intField(data, "deletions"),
		Changes:     intField(data, "changes"),
		IsCreated:   boolField(data, "is_created"),
		IsDeleted:   boolField(data, "is_deleted"),
		IsRenamed:   boolField(data, "is_renamed"),
		IsBinary:    boolField(data, "is_bin"),
		IsSubmodule: boolField(data, "is_submodule"),
	}, true
}

func filterCompareCommits(commits []compareCommit, authorFilter, keyword string) []compareCommit {
	authorFilter = strings.ToLower(strings.TrimSpace(authorFilter))
	keyword = strings.ToLower(strings.TrimSpace(keyword))
	if authorFilter == "" && keyword == "" {
		return append([]compareCommit(nil), commits...)
	}
	filtered := make([]compareCommit, 0, len(commits))
	for _, commit := range commits {
		if authorFilter != "" {
			authorCorpus := strings.ToLower(strings.Join([]string{
				commit.AuthorLogin,
				commit.AuthorName,
				commit.CommitterLogin,
				commit.CommitterName,
			}, " "))
			if !strings.Contains(authorCorpus, authorFilter) {
				continue
			}
		}
		if keyword != "" && !strings.Contains(strings.ToLower(commit.Message), keyword) {
			continue
		}
		filtered = append(filtered, commit)
	}
	return filtered
}

func reverseCompareCommits(commits []compareCommit) {
	for left, right := 0, len(commits)-1; left < right; left, right = left+1, right-1 {
		commits[left], commits[right] = commits[right], commits[left]
	}
}

func uniqueCompareAuthors(commits []compareCommit) []string {
	seen := map[string]bool{}
	authors := make([]string, 0, len(commits))
	for _, commit := range commits {
		author := firstNonEmpty(commit.AuthorLogin, commit.AuthorName, commit.CommitterLogin, commit.CommitterName)
		if author == "" || seen[author] {
			continue
		}
		seen[author] = true
		authors = append(authors, author)
	}
	sort.Strings(authors)
	return authors
}

func summarizeCompareFileTypes(files []compareFile) compareFileTypeSummary {
	summary := compareFileTypeSummary{}
	for _, file := range files {
		switch {
		case file.IsCreated:
			summary.Created++
		case file.IsDeleted:
			summary.Deleted++
		case file.IsRenamed:
			summary.Renamed++
		default:
			summary.Modified++
		}
		if file.IsBinary {
			summary.Binary++
		}
		if file.IsSubmodule {
			summary.Submodule++
		}
	}
	return summary
}

func summarizeCompareBuckets(files []compareFile, bucketFn func(string) string, limit int) []compareBucket {
	type bucketAccum struct {
		files     int
		additions int
		deletions int
		changes   int
	}
	accums := map[string]*bucketAccum{}
	for _, file := range files {
		name := bucketFn(file.Filename)
		if name == "" {
			name = "(none)"
		}
		accum, ok := accums[name]
		if !ok {
			accum = &bucketAccum{}
			accums[name] = accum
		}
		accum.files++
		accum.additions += file.Additions
		accum.deletions += file.Deletions
		accum.changes += file.Changes
	}
	buckets := make([]compareBucket, 0, len(accums))
	for name, accum := range accums {
		buckets = append(buckets, compareBucket{
			Name:      name,
			Files:     accum.files,
			Additions: accum.additions,
			Deletions: accum.deletions,
			Changes:   accum.changes,
		})
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].Changes == buckets[j].Changes {
			return buckets[i].Name < buckets[j].Name
		}
		return buckets[i].Changes > buckets[j].Changes
	})
	if len(buckets) > limit {
		buckets = buckets[:limit]
	}
	return buckets
}

func topCompareFiles(files []compareFile, limit int) []compareFileSummary {
	sortedFiles := append([]compareFile(nil), files...)
	sort.Slice(sortedFiles, func(i, j int) bool {
		if sortedFiles[i].Changes == sortedFiles[j].Changes {
			return sortedFiles[i].Filename < sortedFiles[j].Filename
		}
		return sortedFiles[i].Changes > sortedFiles[j].Changes
	})
	if len(sortedFiles) > limit {
		sortedFiles = sortedFiles[:limit]
	}
	result := make([]compareFileSummary, 0, len(sortedFiles))
	for _, file := range sortedFiles {
		result = append(result, compareFileSummary{
			Filename:  file.Filename,
			Status:    compareFileStatus(file),
			Additions: file.Additions,
			Deletions: file.Deletions,
			Changes:   file.Changes,
		})
	}
	return result
}

func compareFileStatus(file compareFile) string {
	switch {
	case file.IsCreated:
		return "created"
	case file.IsDeleted:
		return "deleted"
	case file.IsRenamed:
		return "renamed"
	default:
		return "modified"
	}
}

func pathGroupForFile(filename string) string {
	filename = strings.ReplaceAll(filename, "\\", "/")
	if !strings.Contains(filename, "/") {
		return "(root)"
	}
	parts := strings.Split(filename, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "(root)"
	}
	return parts[0]
}

func extensionForFile(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return "(none)"
	}
	return ext
}

func subjectFromMessage(message string) string {
	if message == "" {
		return ""
	}
	parts := strings.Split(strings.ReplaceAll(message, "\r\n", "\n"), "\n")
	return strings.TrimSpace(parts[0])
}

func nestedMap(data map[string]interface{}, key string) map[string]interface{} {
	value, _ := data[key].(map[string]interface{})
	return value
}

func stringField(data map[string]interface{}, key string) string {
	if data == nil {
		return ""
	}
	value, _ := data[key].(string)
	return strings.TrimSpace(value)
}

func intField(data map[string]interface{}, keys ...string) int {
	for _, key := range keys {
		switch value := data[key].(type) {
		case float64:
			return int(value)
		case int:
			return value
		case int64:
			return int(value)
		case string:
			if parsed, err := strconv.Atoi(value); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func boolField(data map[string]interface{}, key string) bool {
	if data == nil {
		return false
	}
	value, _ := data[key].(bool)
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func compareTotalsFromFiles(filesResult compareFilesFetchResult) (int, int) {
	if filesResult.TotalAdditions != 0 || filesResult.TotalDeletions != 0 {
		return filesResult.TotalAdditions, filesResult.TotalDeletions
	}
	var additions int
	var deletions int
	for _, file := range filesResult.Files {
		additions += file.Additions
		deletions += file.Deletions
	}
	return additions, deletions
}

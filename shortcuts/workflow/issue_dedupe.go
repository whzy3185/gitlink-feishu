package workflow

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/gitlink-org/gitlink-cli/cmd/cmdutil"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type IssueDedupeInput struct {
	Repository string       `json:"repository"`
	Issues     []IssueInput `json:"issues"`
	Source     string       `json:"source"`
}

type IssueDedupeResult struct {
	Repository       string            `json:"repository"`
	TotalIssues      int               `json:"total_issues"`
	CandidatePairs   int               `json:"candidate_pairs"`
	HighConfidence   int               `json:"high_confidence"`
	MediumConfidence int               `json:"medium_confidence"`
	LowConfidence    int               `json:"low_confidence"`
	Threshold        int               `json:"threshold"`
	Pairs            []IssueDedupePair `json:"pairs"`
	Recommendations  []string          `json:"recommendations"`
	Source           string            `json:"source"`
}

type IssueDedupePair struct {
	Score       int      `json:"score"`
	Confidence  string   `json:"confidence"`
	Primary     IssueRef `json:"primary"`
	Duplicate   IssueRef `json:"duplicate"`
	SharedTerms []string `json:"shared_terms"`
	Reason      string   `json:"reason"`
}

func newIssueDedupeShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "issue-dedupe",
		Description: "Find likely duplicate issues with read-only local similarity rules",
		Flags: []common.Flag{
			{Name: "from", Usage: "Read issues from a JSON file. Supports an array or an object with an issues field"},
			{Name: "state", Short: "s", Usage: "Remote issue state to fetch", Default: "open"},
			{Name: "page", Short: "p", Usage: "Remote issue page", Default: "1"},
			{Name: "limit", Short: "l", Usage: "Maximum issues to analyze", Default: "50"},
			{Name: "threshold", Usage: "Minimum duplicate score from 0 to 100", Default: "55"},
			{Name: "max-pairs", Usage: "Maximum duplicate pairs to output", Default: "20"},
			{Name: "lang", Usage: "Output language: en or zh-CN", Default: langEN},
		},
		Run: runIssueDedupe,
	}
}

func runIssueDedupe(ctx *common.RuntimeContext) error {
	lang := normalizeLang(ctx.Arg("lang"))
	input, threshold, maxPairs, err := collectIssueDedupeInput(ctx)
	if err != nil {
		return err
	}
	result := AnalyzeIssueDedupe(input, threshold, maxPairs, lang)
	format := ctx.Format
	if strings.TrimSpace(cmdutil.Format) == "" {
		format = "table"
	}
	rendered, err := RenderIssueDedupe(result, format, lang)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, rendered)
	return err
}

func collectIssueDedupeInput(ctx *common.RuntimeContext) (IssueDedupeInput, int, int, error) {
	threshold, err := parseIntArg(ctx.Arg("threshold"), 55, "threshold")
	if err != nil {
		return IssueDedupeInput{}, 0, 0, err
	}
	if threshold > 100 {
		return IssueDedupeInput{}, 0, 0, fmt.Errorf("invalid --threshold %q: must be between 0 and 100", ctx.Arg("threshold"))
	}
	maxPairs, err := parseIntArg(ctx.Arg("max-pairs"), 20, "max-pairs")
	if err != nil {
		return IssueDedupeInput{}, 0, 0, err
	}
	if path := strings.TrimSpace(ctx.Arg("from")); path != "" {
		issues, err := readIssueInputs(path)
		if err != nil {
			return IssueDedupeInput{}, 0, 0, err
		}
		limit, err := parseIntArg(ctx.Arg("limit"), 50, "limit")
		if err != nil {
			return IssueDedupeInput{}, 0, 0, err
		}
		return IssueDedupeInput{
			Repository: repositoryFromContext(ctx, ""),
			Issues:     filterIssueInputs(issues, ctx.Arg("state"), limit),
			Source:     "local-json",
		}, threshold, maxPairs, nil
	}
	limit, err := parseIntArg(ctx.Arg("limit"), 50, "limit")
	if err != nil {
		return IssueDedupeInput{}, 0, 0, err
	}
	page, err := parseIntArg(ctx.Arg("page"), 1, "page")
	if err != nil {
		return IssueDedupeInput{}, 0, 0, err
	}
	state := strings.TrimSpace(ctx.Arg("state"))
	if state == "" {
		state = "open"
	}
	issues, err := FetchIssuesForTriage(ctx, TriageFetchOptions{
		State: state,
		Limit: limit,
		Page:  page,
	})
	if err != nil {
		return IssueDedupeInput{}, 0, 0, fmt.Errorf("workflow +issue-dedupe remote mode requires --owner and --repo or a Git remote: %w", err)
	}
	return IssueDedupeInput{
		Repository: repositoryFromContext(ctx, ""),
		Issues:     issues,
		Source:     "remote-read-only-fetch",
	}, threshold, maxPairs, nil
}

func AnalyzeIssueDedupe(input IssueDedupeInput, threshold int, maxPairs int, lang string) IssueDedupeResult {
	lang = normalizeLang(lang)
	if threshold < 0 {
		threshold = 0
	}
	if threshold > 100 {
		threshold = 100
	}
	repository := strings.TrimSpace(input.Repository)
	if repository == "" {
		repository = "local"
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = "local"
	}
	fingerprints := make([]issueFingerprint, 0, len(input.Issues))
	for _, issue := range input.Issues {
		fp := buildIssueFingerprint(issue)
		if len(fp.Terms) > 0 {
			fingerprints = append(fingerprints, fp)
		}
	}
	pairs := []IssueDedupePair{}
	for i := 0; i < len(fingerprints); i++ {
		for j := i + 1; j < len(fingerprints); j++ {
			score, shared := scoreIssueSimilarity(fingerprints[i], fingerprints[j])
			if score < threshold {
				continue
			}
			pairs = append(pairs, buildIssueDedupePair(fingerprints[i], fingerprints[j], score, shared, lang))
		}
	}
	sort.SliceStable(pairs, func(i, j int) bool {
		if pairs[i].Score != pairs[j].Score {
			return pairs[i].Score > pairs[j].Score
		}
		return pairs[i].Primary.Number < pairs[j].Primary.Number
	})
	if maxPairs > 0 && len(pairs) > maxPairs {
		pairs = pairs[:maxPairs]
	}
	result := IssueDedupeResult{
		Repository:     repository,
		TotalIssues:    len(input.Issues),
		CandidatePairs: len(pairs),
		Threshold:      threshold,
		Pairs:          pairs,
		Source:         source,
	}
	for _, pair := range pairs {
		switch pair.Confidence {
		case "high":
			result.HighConfidence++
		case "medium":
			result.MediumConfidence++
		default:
			result.LowConfidence++
		}
	}
	result.Recommendations = issueDedupeRecommendations(result, lang)
	return result
}

type issueFingerprint struct {
	Issue IssueInput
	Terms map[string]int
}

func buildIssueFingerprint(issue IssueInput) issueFingerprint {
	terms := map[string]int{}
	for _, token := range tokenizeIssueText(issue.Title + " " + issue.Body) {
		terms[token]++
	}
	for _, label := range issue.Labels {
		for _, token := range tokenizeIssueText(label) {
			terms[token] += 2
		}
	}
	return issueFingerprint{Issue: issue, Terms: terms}
}

func tokenizeIssueText(value string) []string {
	value = strings.ToLower(value)
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte(' ')
		}
	}
	raw := strings.Fields(b.String())
	tokens := make([]string, 0, len(raw))
	for _, token := range raw {
		if len([]rune(token)) < 3 || issueDedupeStopWords[token] {
			continue
		}
		tokens = append(tokens, token)
	}
	return tokens
}

var issueDedupeStopWords = map[string]bool{
	"the": true, "and": true, "for": true, "with": true, "when": true, "from": true, "this": true, "that": true,
	"can": true, "not": true, "cannot": true, "should": true, "would": true, "could": true, "please": true,
	"issue": true, "bug": true, "error": true, "failed": true, "failure": true, "using": true, "after": true,
}

func scoreIssueSimilarity(a, b issueFingerprint) (int, []string) {
	intersectionWeight := 0
	unionWeight := 0
	shared := []string{}
	seen := map[string]bool{}
	for term, weightA := range a.Terms {
		weightB := b.Terms[term]
		if weightB > 0 {
			intersectionWeight += minInt(weightA, weightB)
			shared = append(shared, term)
		}
		unionWeight += maxIssueDedupeInt(weightA, weightB)
		seen[term] = true
	}
	for term, weightB := range b.Terms {
		if seen[term] {
			continue
		}
		unionWeight += weightB
	}
	if unionWeight == 0 {
		return 0, nil
	}
	sort.Strings(shared)
	score := int(math.Round(float64(intersectionWeight) / float64(unionWeight) * 100))
	if sameNonZeroIssueType(a.Issue, b.Issue) {
		score += 8
	}
	if score > 100 {
		score = 100
	}
	return score, limitIssueDedupeTerms(shared, 8)
}

func sameNonZeroIssueType(a, b IssueInput) bool {
	typeA := AnalyzeIssue(a, langEN).DetectedType
	typeB := AnalyzeIssue(b, langEN).DetectedType
	return typeA != IssueTypeUnknown && typeA == typeB
}

func buildIssueDedupePair(a, b issueFingerprint, score int, shared []string, lang string) IssueDedupePair {
	confidence := "low"
	if score >= 80 {
		confidence = "high"
	} else if score >= 65 {
		confidence = "medium"
	}
	primary := issueRefFromInput(a.Issue)
	duplicate := issueRefFromInput(b.Issue)
	if duplicate.Number > 0 && (primary.Number == 0 || duplicate.Number < primary.Number) {
		primary, duplicate = duplicate, primary
	}
	reason := fmt.Sprintf("shared terms: %s", strings.Join(shared, ", "))
	if normalizeLang(lang) == langZH {
		reason = fmt.Sprintf("共享关键词：%s", strings.Join(shared, "、"))
	}
	return IssueDedupePair{
		Score:       score,
		Confidence:  confidence,
		Primary:     primary,
		Duplicate:   duplicate,
		SharedTerms: shared,
		Reason:      reason,
	}
}

func issueRefFromInput(issue IssueInput) IssueRef {
	return IssueRef{
		ID:     issue.ID,
		Number: issue.Number,
		Title:  issue.Title,
		URL:    issue.URL,
		Author: issue.Author,
		State:  issue.State,
	}
}

func issueDedupeRecommendations(result IssueDedupeResult, lang string) []string {
	zh := normalizeLang(lang) == langZH
	if result.CandidatePairs == 0 {
		if zh {
			return []string{"未发现达到阈值的重复候选，可以按常规 Issue 流程继续处理。"}
		}
		return []string{"No duplicate candidates reached the threshold; continue with normal issue triage."}
	}
	recs := []string{}
	if result.HighConfidence > 0 {
		if zh {
			recs = append(recs, "优先人工确认高置信候选，必要时在后创建的 Issue 中链接原始 Issue。")
		} else {
			recs = append(recs, "Review high-confidence candidates first and link newer issues to the original when appropriate.")
		}
	}
	if zh {
		recs = append(recs, "命令只提供候选对，不会自动关闭、评论或修改 Issue。")
	} else {
		recs = append(recs, "The command only reports candidates; it never closes, comments on, or edits issues.")
	}
	return recs
}

func RenderIssueDedupe(result IssueDedupeResult, format string, lang string) (string, error) {
	var buf bytes.Buffer
	switch normalizeFormat(format) {
	case "json":
		if err := writeJSON(&buf, result); err != nil {
			return "", err
		}
	case "markdown":
		if err := writeIssueDedupeMarkdown(&buf, result, lang); err != nil {
			return "", err
		}
	case "table":
		if err := writeIssueDedupeTable(&buf, result); err != nil {
			return "", err
		}
	default:
		return "", fmt.Errorf("unsupported workflow output format %q", format)
	}
	return buf.String(), nil
}

func writeIssueDedupeMarkdown(buf *bytes.Buffer, result IssueDedupeResult, lang string) error {
	title := "Issue Duplicate Candidates"
	if normalizeLang(lang) == langZH {
		title = "Issue 重复候选"
	}
	if _, err := fmt.Fprintf(buf, "# %s\n\n", title); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(buf, "- Repository: `%s`\n- Issues analyzed: `%d`\n- Candidate pairs: `%d`\n- Threshold: `%d`\n- Source: `%s`\n\n",
		result.Repository, result.TotalIssues, result.CandidatePairs, result.Threshold, result.Source); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf, "## Candidates"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if len(result.Pairs) == 0 {
		_, err := fmt.Fprintln(buf, "- No duplicate candidates.")
		return err
	}
	for _, pair := range result.Pairs {
		if _, err := fmt.Fprintf(buf, "- `%s` score `%d`: #%d %s -> #%d %s\n",
			pair.Confidence, pair.Score, pair.Primary.Number, pair.Primary.Title, pair.Duplicate.Number, pair.Duplicate.Title); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(buf, "  - %s\n", pair.Reason); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(buf); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(buf, "## Recommendations"); err != nil {
		return err
	}
	for _, rec := range result.Recommendations {
		if _, err := fmt.Fprintf(buf, "- %s\n", rec); err != nil {
			return err
		}
	}
	return nil
}

func writeIssueDedupeTable(buf *bytes.Buffer, result IssueDedupeResult) error {
	tw := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "SCORE\tCONFIDENCE\tPRIMARY\tDUPLICATE\tSHARED_TERMS"); err != nil {
		return err
	}
	for _, pair := range result.Pairs {
		if _, err := fmt.Fprintf(tw, "%d\t%s\t#%d %s\t#%d %s\t%s\n",
			pair.Score,
			pair.Confidence,
			pair.Primary.Number,
			truncateTableText(pair.Primary.Title, 48),
			pair.Duplicate.Number,
			truncateTableText(pair.Duplicate.Title, 48),
			strings.Join(pair.SharedTerms, ","),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

func limitIssueDedupeTerms(values []string, limit int) []string {
	if limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func maxIssueDedupeInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

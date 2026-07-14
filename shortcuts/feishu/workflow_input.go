package feishu

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"

	"github.com/gitlink-org/gitlink-cli/shortcuts/workflow"
)

func readWorkflowReport(path string, lang string) (workflow.RepoReportResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return workflow.RepoReportResult{}, fmt.Errorf("read workflow JSON: %w", err)
	}
	data, err = normalizeJSONBytes(data)
	if err != nil {
		return workflow.RepoReportResult{}, err
	}
	data, err = unwrapWorkflowJSON(data)
	if err != nil {
		return workflow.RepoReportResult{}, err
	}

	var result workflow.RepoReportResult
	if err := json.Unmarshal(data, &result); err == nil && looksLikeRepoReportResult(result) {
		return normalizeReportResult(result), nil
	}

	var input workflow.RepoReportInput
	if err := json.Unmarshal(data, &input); err == nil && looksLikeRepoReportInput(input) {
		return workflow.AnalyzeRepoReport(input, lang), nil
	}

	return workflow.RepoReportResult{}, fmt.Errorf("parse workflow JSON: expected workflow RepoReportResult or RepoReportInput")
}

func normalizeJSONBytes(data []byte) ([]byte, error) {
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		return data[3:], nil
	}
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xFE {
		return decodeUTF16(data[2:], true)
	}
	if len(data) >= 2 && data[0] == 0xFE && data[1] == 0xFF {
		return decodeUTF16(data[2:], false)
	}
	return data, nil
}

func decodeUTF16(data []byte, littleEndian bool) ([]byte, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("parse workflow JSON: invalid UTF-16 byte length")
	}
	words := make([]uint16, 0, len(data)/2)
	for i := 0; i < len(data); i += 2 {
		if littleEndian {
			words = append(words, uint16(data[i])|uint16(data[i+1])<<8)
		} else {
			words = append(words, uint16(data[i])<<8|uint16(data[i+1]))
		}
	}
	return []byte(string(utf16.Decode(words))), nil
}

func unwrapWorkflowJSON(data []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse workflow JSON: %w", err)
	}
	for _, key := range []string{"data", "repo_report", "report"} {
		if value, ok := raw[key]; ok && len(value) > 0 && string(value) != "null" {
			return value, nil
		}
	}
	return data, nil
}

func looksLikeRepoReportResult(result workflow.RepoReportResult) bool {
	return strings.TrimSpace(result.Repository) != "" &&
		(strings.TrimSpace(result.RiskLevel) != "" ||
			result.ReportScore != 0 ||
			result.IssueSummary.Total != 0 ||
			result.PRSummary.Total != 0 ||
			len(result.Recommendations) > 0)
}

func looksLikeRepoReportInput(input workflow.RepoReportInput) bool {
	return strings.TrimSpace(input.Repository) != "" ||
		input.Health != nil ||
		len(input.Issues) > 0 ||
		len(input.PullRequests) > 0
}

func normalizeReportResult(result workflow.RepoReportResult) workflow.RepoReportResult {
	if strings.TrimSpace(result.Source) == "" {
		result.Source = "workflow-json"
	}
	if result.IssueSummary.ByType == nil {
		result.IssueSummary.ByType = map[string]int{}
	}
	if result.IssueSummary.ByPriority == nil {
		result.IssueSummary.ByPriority = map[string]int{}
	}
	if result.PRSummary.ByType == nil {
		result.PRSummary.ByType = map[string]int{}
	}
	if result.PRSummary.ByRisk == nil {
		result.PRSummary.ByRisk = map[string]int{}
	}
	return result
}

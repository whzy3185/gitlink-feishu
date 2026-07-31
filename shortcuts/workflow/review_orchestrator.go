package workflow

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gitlink-org/gitlink-cli/internal/collab"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	reviewAgentPlanSchema       = "review.agent-plan/v1"
	reviewAgentAssessmentSchema = "review.agent-assessment/v1"
	reviewWarroomSchema         = "review.warroom/v1"
)

type ReviewAgentPlan struct {
	SchemaVersion         string              `json:"schema_version"`
	RunID                 string              `json:"run_id"`
	Repository            string              `json:"repository"`
	PRNumber              int                 `json:"pr_number"`
	HeadSHA               string              `json:"head_sha"`
	SourceFingerprint     string              `json:"source_fingerprint"`
	Mode                  string              `json:"mode"`
	ChangedFiles          []ReviewChangedFile `json:"changed_files"`
	Tasks                 []ReviewAgentTask   `json:"tasks"`
	MaxConcurrency        int                 `json:"max_concurrency"`
	HumanDecisionRequired bool                `json:"human_decision_required"`
	GitLinkWrites         int                 `json:"gitlink_writes"`
	Capabilities          []ReviewCapability  `json:"capabilities"`
	CreatedAt             string              `json:"created_at"`
}

type ReviewChangedFile struct {
	Path        string `json:"path"`
	ChangeKind  string `json:"change_kind"`
	Fingerprint string `json:"fingerprint"`
}

type ReviewAgentTask struct {
	TaskID          string   `json:"task_id"`
	Role            string   `json:"role"`
	Objective       string   `json:"objective"`
	FileScopes      []string `json:"file_scopes"`
	RequiredOutputs []string `json:"required_outputs"`
	ReadOnly        bool     `json:"read_only"`
}

type ReviewCapability struct {
	Action  string `json:"action"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason"`
}

type ReviewAgentAssessment struct {
	SchemaVersion string               `json:"schema_version"`
	RunID         string               `json:"run_id"`
	TaskID        string               `json:"task_id"`
	Role          string               `json:"role"`
	HeadSHA       string               `json:"head_sha"`
	Status        string               `json:"status"`
	Findings      []ReviewAgentFinding `json:"findings"`
	Unknowns      []string             `json:"unknowns,omitempty"`
	CompletedAt   string               `json:"completed_at"`
}

type ReviewAgentFinding struct {
	Severity   string `json:"severity"`
	Path       string `json:"path,omitempty"`
	Line       int    `json:"line,omitempty"`
	Summary    string `json:"summary"`
	Evidence   string `json:"evidence"`
	Confidence string `json:"confidence"`
}

type ReviewAgentSynthesis struct {
	SchemaVersion         string               `json:"schema_version"`
	RunID                 string               `json:"run_id"`
	HeadSHA               string               `json:"head_sha"`
	Status                string               `json:"status"`
	AssessmentsReceived   int                  `json:"assessments_received"`
	AssessmentsExpected   int                  `json:"assessments_expected"`
	Findings              []ReviewAgentFinding `json:"findings"`
	Unknowns              []string             `json:"unknowns,omitempty"`
	Conflicts             []string             `json:"conflicts,omitempty"`
	Recommendation        string               `json:"recommendation"`
	HumanDecisionRequired bool                 `json:"human_decision_required"`
	GitLinkWrites         int                  `json:"gitlink_writes"`
}

type ReviewWarroom struct {
	SchemaVersion         string            `json:"schema_version"`
	GeneratedAt           string            `json:"generated_at"`
	Repositories          []string          `json:"repositories"`
	Items                 []collab.WorkItem `json:"items"`
	CompleteItems         int               `json:"complete_items"`
	PartialItems          int               `json:"partial_items"`
	HumanDecisionRequired bool              `json:"human_decision_required"`
	GitLinkWrites         int               `json:"gitlink_writes"`
}

func newReviewOrchestrateShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-orchestrate",
		Description: "Build a deterministic, read-only multi-agent Review plan from review.context/v1",
		Flags: []common.Flag{
			{Name: "from", Usage: "Current review.context/v1 JSON", Required: true},
			{Name: "previous", Usage: "Optional previous review.context/v1 JSON for incremental Review"},
			{Name: "max-agents", Usage: "Maximum concurrent specialist agents", Default: "3"},
		},
		Run: func(runtime *common.RuntimeContext) error {
			current, err := readReviewContextInput(runtime.Arg("from"))
			if err != nil {
				return err
			}
			var previous *ReviewContext
			if path := strings.TrimSpace(runtime.Arg("previous")); path != "" {
				loaded, loadErr := readReviewContextInput(path)
				if loadErr != nil {
					return loadErr
				}
				previous = &loaded
			}
			maxAgents, err := strconv.Atoi(firstNonEmptyWorkflow(runtime.Arg("max-agents"), "3"))
			if err != nil || maxAgents < 1 || maxAgents > 8 {
				return fmt.Errorf("--max-agents must be between 1 and 8")
			}
			return runtime.OutputData(BuildReviewAgentPlan(current, previous, maxAgents, time.Now().UTC()))
		},
	}
}

func newReviewSynthesizeShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-synthesize",
		Description: "Validate and synthesize specialist assessments without making the owner decision",
		Flags: []common.Flag{
			{Name: "plan", Usage: "review.agent-plan/v1 JSON", Required: true},
			{Name: "assessments", Usage: "Comma-separated review.agent-assessment/v1 JSON files", Required: true},
		},
		Run: func(runtime *common.RuntimeContext) error {
			plan, err := readReviewAgentPlan(runtime.Arg("plan"))
			if err != nil {
				return err
			}
			assessments, err := readReviewAgentAssessments(runtime.Arg("assessments"))
			if err != nil {
				return err
			}
			return runtime.OutputData(SynthesizeReviewAssessments(plan, assessments))
		},
	}
}

func newReviewWarroomShortcut() *common.Shortcut {
	return &common.Shortcut{
		Name:        "review-warroom",
		Description: "Build a cross-repository owner view from one or more review.context/v1 files",
		Flags: []common.Flag{
			{Name: "from", Usage: "Comma-separated review.context/v1 JSON files", Required: true},
			{Name: "collaboration", Usage: "Optional comma-separated canonical P2 WorkItem or collaboration bundle JSON files"},
		},
		Run: func(runtime *common.RuntimeContext) error {
			contexts := []ReviewContext{}
			for _, path := range splitReviewPaths(runtime.Arg("from")) {
				loaded, err := readReviewContextInput(path)
				if err != nil {
					return err
				}
				contexts = append(contexts, loaded)
			}
			if len(contexts) == 0 {
				return fmt.Errorf("--from requires at least one Review Context file")
			}
			collaborationItems, err := readReviewCollaborationItems(runtime.Arg("collaboration"))
			if err != nil {
				return err
			}
			return runtime.OutputData(BuildReviewWarroomWithCollaboration(
				contexts,
				collaborationItems,
				time.Now().UTC(),
			))
		},
	}
}

func BuildReviewAgentPlan(
	current ReviewContext,
	previous *ReviewContext,
	maxAgents int,
	now time.Time,
) ReviewAgentPlan {
	changed := calculateReviewChangedFiles(current, previous)
	mode := "full"
	if previous != nil {
		mode = "incremental"
	}
	seed := strings.Join([]string{
		current.Repository,
		strconv.Itoa(current.PullRequest),
		current.CurrentHeadSHA,
		current.WorkItem.SourceFingerprint,
		mode,
	}, "\x00")
	digest := sha256.Sum256([]byte(seed))
	plan := ReviewAgentPlan{
		SchemaVersion:         reviewAgentPlanSchema,
		RunID:                 "review-run-" + hex.EncodeToString(digest[:8]),
		Repository:            current.Repository,
		PRNumber:              current.PullRequest,
		HeadSHA:               current.CurrentHeadSHA,
		SourceFingerprint:     current.WorkItem.SourceFingerprint,
		Mode:                  mode,
		ChangedFiles:          changed,
		MaxConcurrency:        maxAgents,
		HumanDecisionRequired: true,
		GitLinkWrites:         0,
		Capabilities: []ReviewCapability{
			{Action: "read_context", Enabled: true, Reason: "stable GET-only Review context"},
			{Action: "produce_findings", Enabled: true, Reason: "evidence-backed local assessment"},
			{Action: "common_review_action_plan", Enabled: true, Reason: "requires explicit identity and confirmation"},
			{Action: "approved_or_rejected_review", Enabled: false, Reason: "reserved for future owner-governed policy"},
			{Action: "line_comment", Enabled: false, Reason: "no stable write and reconciliation contract"},
			{Action: "reviewer_management", Enabled: false, Reason: "organization permission policy not implemented"},
			{Action: "merge", Enabled: false, Reason: "final decision remains with the repository owner"},
		},
		CreatedAt: now.UTC().Format(time.RFC3339Nano),
	}
	scopes := changedFilePaths(changed)
	plan.Tasks = append(plan.Tasks,
		newReviewAgentTask(plan.RunID, "correctness", "Find behavior regressions and contract violations", scopes),
		newReviewAgentTask(plan.RunID, "tests", "Evaluate regression coverage and reproducibility", scopes),
	)
	if hasSensitiveReviewPath(scopes) {
		plan.Tasks = append(plan.Tasks, newReviewAgentTask(
			plan.RunID,
			"security",
			"Review authentication, authorization, secret handling, and unsafe execution boundaries",
			scopes,
		))
	}
	if hasDocumentationReviewPath(scopes) {
		plan.Tasks = append(plan.Tasks, newReviewAgentTask(
			plan.RunID,
			"contributor_experience",
			"Check contributor instructions, compatibility, and actionable feedback quality",
			scopes,
		))
	}
	plan.Tasks = append(plan.Tasks, newReviewAgentTask(
		plan.RunID,
		"integration",
		"Check cross-component behavior, rollout safety, and owner-facing operational impact",
		scopes,
	))
	return plan
}

func SynthesizeReviewAssessments(
	plan ReviewAgentPlan,
	assessments []ReviewAgentAssessment,
) ReviewAgentSynthesis {
	result := ReviewAgentSynthesis{
		SchemaVersion:         "review.agent-synthesis/v1",
		RunID:                 plan.RunID,
		HeadSHA:               plan.HeadSHA,
		AssessmentsExpected:   len(plan.Tasks),
		HumanDecisionRequired: true,
		GitLinkWrites:         0,
		Recommendation:        "human_decision_required",
	}
	taskRoles := map[string]string{}
	for _, task := range plan.Tasks {
		taskRoles[task.TaskID] = task.Role
	}
	seen := map[string]bool{}
	for _, assessment := range assessments {
		if assessment.SchemaVersion != reviewAgentAssessmentSchema ||
			assessment.RunID != plan.RunID ||
			assessment.HeadSHA != plan.HeadSHA {
			result.Conflicts = append(result.Conflicts, "assessment rejected because schema, run, or head did not match")
			continue
		}
		if validationError := validateReviewAgentAssessment(assessment); validationError != "" {
			result.Conflicts = append(result.Conflicts, "assessment rejected: "+validationError)
			continue
		}
		role, exists := taskRoles[assessment.TaskID]
		if !exists || role != assessment.Role || seen[assessment.TaskID] {
			result.Conflicts = append(result.Conflicts, "assessment rejected because task ownership was invalid or duplicated")
			continue
		}
		seen[assessment.TaskID] = true
		result.AssessmentsReceived++
		result.Findings = append(result.Findings, assessment.Findings...)
		result.Unknowns = append(result.Unknowns, assessment.Unknowns...)
	}
	sort.SliceStable(result.Findings, func(i, j int) bool {
		return reviewSeverityRank(result.Findings[i].Severity) >
			reviewSeverityRank(result.Findings[j].Severity)
	})
	switch {
	case len(result.Conflicts) > 0 || result.AssessmentsReceived < result.AssessmentsExpected:
		result.Status = "incomplete"
	case hasBlockingReviewFinding(result.Findings):
		result.Status = "findings_require_changes"
	default:
		result.Status = "ready_for_owner_review"
	}
	return result
}

func validateReviewAgentAssessment(assessment ReviewAgentAssessment) string {
	if assessment.Status != "completed" {
		return "status must be completed"
	}
	if _, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(assessment.CompletedAt)); err != nil {
		return "completed_at must be an RFC3339 timestamp"
	}
	for _, finding := range assessment.Findings {
		if reviewSeverityRank(finding.Severity) == 0 {
			return "finding severity must be critical, high, medium, or low"
		}
		if strings.TrimSpace(finding.Summary) == "" ||
			strings.TrimSpace(finding.Evidence) == "" ||
			strings.TrimSpace(finding.Confidence) == "" {
			return "each finding requires summary, evidence, and confidence"
		}
		if finding.Line < 0 {
			return "finding line cannot be negative"
		}
	}
	for _, unknown := range assessment.Unknowns {
		if strings.TrimSpace(unknown) == "" {
			return "unknown entries cannot be empty"
		}
	}
	return ""
}

func BuildReviewWarroom(contexts []ReviewContext, now time.Time) ReviewWarroom {
	return BuildReviewWarroomWithCollaboration(contexts, nil, now)
}

func BuildReviewWarroomWithCollaboration(
	contexts []ReviewContext,
	collaborationItems []collab.WorkItem,
	now time.Time,
) ReviewWarroom {
	result := ReviewWarroom{
		SchemaVersion:         reviewWarroomSchema,
		GeneratedAt:           now.UTC().Format(time.RFC3339Nano),
		HumanDecisionRequired: true,
		GitLinkWrites:         0,
	}
	repositories := map[string]bool{}
	collaborationByKey := map[string]collab.WorkItem{}
	for _, item := range collaborationItems {
		if item.PRKey != "" {
			collaborationByKey[item.PRKey] = item
		}
	}
	for _, reviewContext := range contexts {
		repositories[reviewContext.Repository] = true
		collaboration := collaborationByKey[reviewContext.WorkItem.PRKey]
		collaborationStatus := collaboration.CollaborationStatus
		if collaborationStatus == "" {
			collaborationStatus = "unassigned"
		}
		item := collab.WorkItem{
			SchemaVersion:       collab.WorkItemSchema,
			PRKey:               reviewContext.WorkItem.PRKey,
			Repository:          reviewContext.Repository,
			PRNumber:            reviewContext.PullRequest,
			GitLinkURL:          reviewContext.WorkItem.GitLinkURL,
			GitLinkState:        reviewContext.WorkItem.GitLinkState,
			ReviewStage:         reviewContext.WorkItem.ReviewStage,
			Decision:            reviewContext.Summary.Decision,
			CollectionStatus:    reviewContext.CollectionStatus,
			HeadSHA:             reviewContext.CurrentHeadSHA,
			SourceFingerprint:   reviewContext.WorkItem.SourceFingerprint,
			AssignedTo:          collaboration.AssignedTo,
			CollaborationStatus: collaborationStatus,
			DueAt:               collaboration.DueAt,
			Archived:            collaboration.Archived || reviewContext.WorkItem.GitLinkState == "merged" || reviewContext.WorkItem.GitLinkState == "closed",
			NextStep:            firstNonEmptyWorkflow(collaboration.NextStep, reviewContext.WorkItem.RecommendedNextStep),
			UpdatedAt:           reviewContext.WorkItem.GeneratedAt,
		}
		result.Items = append(result.Items, item)
		if reviewContext.Partial || reviewContext.CollectionStatus != "complete" {
			result.PartialItems++
		} else {
			result.CompleteItems++
		}
	}
	for repository := range repositories {
		result.Repositories = append(result.Repositories, repository)
	}
	sort.Strings(result.Repositories)
	sort.SliceStable(result.Items, func(i, j int) bool {
		left := warroomReviewRank(contexts, result.Items[i].PRKey)
		right := warroomReviewRank(contexts, result.Items[j].PRKey)
		if left != right {
			return left > right
		}
		return result.Items[i].PRKey < result.Items[j].PRKey
	})
	return result
}

func readReviewCollaborationItems(value string) ([]collab.WorkItem, error) {
	items := []collab.WorkItem{}
	for _, path := range splitReviewPaths(value) {
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var item collab.WorkItem
		if err := json.Unmarshal(payload, &item); err == nil && item.SchemaVersion == collab.WorkItemSchema {
			if err := item.Validate(); err != nil {
				return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
			}
			items = append(items, item)
			continue
		}
		var wrapper struct {
			Canonical collab.WorkItem `json:"canonical"`
		}
		if err := json.Unmarshal(payload, &wrapper); err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		if err := wrapper.Canonical.Validate(); err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		items = append(items, wrapper.Canonical)
	}
	return items, nil
}

func calculateReviewChangedFiles(current ReviewContext, previous *ReviewContext) []ReviewChangedFile {
	currentFiles := reviewFileFingerprints(current.Files)
	previousFiles := map[string]string{}
	if previous != nil {
		previousFiles = reviewFileFingerprints(previous.Files)
	}
	paths := map[string]bool{}
	for path := range currentFiles {
		paths[path] = true
	}
	for path := range previousFiles {
		paths[path] = true
	}
	result := []ReviewChangedFile{}
	for path := range paths {
		changeKind := "unchanged"
		switch {
		case previous == nil:
			changeKind = "current_patchset"
		case currentFiles[path] == "":
			changeKind = "removed"
		case previousFiles[path] == "":
			changeKind = "added"
		case currentFiles[path] != previousFiles[path]:
			changeKind = "modified"
		default:
			continue
		}
		result = append(result, ReviewChangedFile{
			Path:        path,
			ChangeKind:  changeKind,
			Fingerprint: firstNonEmptyWorkflow(currentFiles[path], previousFiles[path]),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func reviewFileFingerprints(files []map[string]interface{}) map[string]string {
	result := map[string]string{}
	for index, file := range files {
		path := firstPRString(file, "filename", "path", "new_path", "name")
		if path == "" {
			path = fmt.Sprintf("unknown-file-%d", index+1)
		}
		payload, _ := json.Marshal(file)
		digest := sha256.Sum256(payload)
		result[path] = hex.EncodeToString(digest[:8])
	}
	return result
}

func newReviewAgentTask(runID, role, objective string, scopes []string) ReviewAgentTask {
	digest := sha256.Sum256([]byte(runID + "\x00" + role))
	return ReviewAgentTask{
		TaskID:     "agent-task-" + hex.EncodeToString(digest[:6]),
		Role:       role,
		Objective:  objective,
		FileScopes: append([]string(nil), scopes...),
		RequiredOutputs: []string{
			"severity",
			"path and tight line when available",
			"evidence",
			"confidence",
			"unknowns",
		},
		ReadOnly: true,
	}
}

func changedFilePaths(files []ReviewChangedFile) []string {
	result := make([]string, 0, len(files))
	for _, file := range files {
		result = append(result, file.Path)
	}
	return result
}

func hasSensitiveReviewPath(paths []string) bool {
	for _, path := range paths {
		lower := strings.ToLower(path)
		if strings.Contains(lower, "auth") || strings.Contains(lower, "crypto") ||
			strings.Contains(lower, "secret") || strings.Contains(lower, "token") ||
			strings.Contains(lower, "client") || strings.Contains(lower, "gateway") {
			return true
		}
	}
	return false
}

func hasDocumentationReviewPath(paths []string) bool {
	for _, path := range paths {
		lower := strings.ToLower(path)
		if strings.HasSuffix(lower, ".md") || strings.Contains(lower, "docs/") ||
			strings.Contains(lower, "readme") || strings.Contains(lower, "contributing") {
			return true
		}
	}
	return false
}

func hasBlockingReviewFinding(findings []ReviewAgentFinding) bool {
	for _, finding := range findings {
		if reviewSeverityRank(finding.Severity) >= 3 {
			return true
		}
	}
	return false
}

func reviewSeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func warroomReviewRank(contexts []ReviewContext, prKey string) int {
	for _, item := range contexts {
		if item.WorkItem.PRKey != prKey {
			continue
		}
		switch item.WorkItem.Priority {
		case "high":
			return 3
		case "medium":
			return 2
		case "low":
			return 1
		}
	}
	return 0
}

func readReviewAgentPlan(path string) (ReviewAgentPlan, error) {
	var plan ReviewAgentPlan
	payload, err := os.ReadFile(strings.TrimSpace(path))
	if err != nil {
		return plan, err
	}
	if err := json.Unmarshal(payload, &plan); err != nil {
		return plan, err
	}
	if plan.SchemaVersion != reviewAgentPlanSchema || plan.RunID == "" || plan.HeadSHA == "" {
		return plan, fmt.Errorf("invalid review.agent-plan/v1")
	}
	return plan, nil
}

func readReviewAgentAssessments(value string) ([]ReviewAgentAssessment, error) {
	result := []ReviewAgentAssessment{}
	for _, path := range splitReviewPaths(value) {
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		var assessment ReviewAgentAssessment
		if err := json.Unmarshal(payload, &assessment); err != nil {
			return nil, fmt.Errorf("parse %s: %w", filepath.Base(path), err)
		}
		result = append(result, assessment)
	}
	return result, nil
}

func splitReviewPaths(value string) []string {
	result := []string{}
	for _, path := range strings.Split(value, ",") {
		if path = strings.TrimSpace(path); path != "" {
			result = append(result, path)
		}
	}
	return result
}

func firstNonEmptyWorkflow(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

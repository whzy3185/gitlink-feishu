package workflow

import "time"

const (
	langEN = "en"
	langZH = "zh-CN"
)

const (
	IssueTypeBug         = "bug"
	IssueTypeFeature     = "feature"
	IssueTypeQuestion    = "question"
	IssueTypeDocs        = "docs"
	IssueTypeCI          = "ci"
	IssueTypeSecurity    = "security"
	IssueTypePerformance = "performance"
	IssueTypeRefactor    = "refactor"
	IssueTypeUnknown     = "unknown"
)

const (
	PriorityP0 = "P0"
	PriorityP1 = "P1"
	PriorityP2 = "P2"
	PriorityP3 = "P3"
)

const (
	RiskSecuritySensitive     = "security_sensitive"
	RiskPossibleSecretLeak    = "possible_secret_leak"
	RiskInstallationBlocker   = "installation_blocker"
	RiskAuthenticationBlocker = "authentication_blocker"
	RiskCIBlocker             = "ci_blocker"
	RiskInsufficientInfo      = "insufficient_information"
)

const (
	ActionRequestMoreInfo     = "request_more_info"
	ActionPrioritizeImmediate = "prioritize_immediately"
	ActionScheduleFix         = "schedule_fix"
	ActionConvertToDiscussion = "convert_to_discussion"
	ActionUpdateDocs          = "update_docs"
	ActionInvestigateCI       = "investigate_ci"
	ActionReviewSecurity      = "review_security"
)

type IssueInput struct {
	ID            string    `json:"id"`
	Number        int       `json:"number"`
	Title         string    `json:"title"`
	Body          string    `json:"body"`
	State         string    `json:"state"`
	Author        string    `json:"author"`
	URL           string    `json:"url"`
	Labels        []string  `json:"labels"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	CommentsCount int       `json:"comments_count"`
}

type IssueRef struct {
	ID     string `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	Author string `json:"author"`
	State  string `json:"state"`
}

type TriageResult struct {
	Issue              IssueRef `json:"issue"`
	DetectedType       string   `json:"detected_type"`
	Priority           string   `json:"priority"`
	Confidence         int      `json:"confidence"`
	SuggestedLabels    []string `json:"suggested_labels"`
	MissingInformation []string `json:"missing_information"`
	RiskFlags          []string `json:"risk_flags"`
	RecommendedAction  string   `json:"recommended_action"`
	SuggestedComment   string   `json:"suggested_comment"`
	Reasoning          []string `json:"reasoning"`
	MatchedRules       []string `json:"matched_rules"`
}

type HealthInput struct {
	Repository          string `json:"repository"`
	OpenIssues          int    `json:"open_issues"`
	OpenPRs             int    `json:"open_prs"`
	StaleIssues         int    `json:"stale_issues"`
	StalePRs            int    `json:"stale_prs"`
	RecentActivityKnown bool   `json:"recent_activity_known"`
	RecentActivityDays  int    `json:"recent_activity_days"`
	ReleaseKnown        bool   `json:"release_known"`
	HasRecentRelease    bool   `json:"has_recent_release"`
	CIKnown             bool   `json:"ci_known"`
	CIPassing           bool   `json:"ci_passing"`
	HasReadme           bool   `json:"has_readme"`
	HasLicense          bool   `json:"has_license"`
	HasContributing     bool   `json:"has_contributing"`
	AgentReadinessKnown bool   `json:"agent_readiness_known"`
	AgentReadinessScore int    `json:"agent_readiness_score"`
}

type HealthResult struct {
	Repository      string         `json:"repository"`
	HealthScore     int            `json:"health_score"`
	RiskLevel       string         `json:"risk_level"`
	Metrics         []HealthMetric `json:"metrics"`
	Recommendations []string       `json:"recommendations"`
	ScoringNotes    []ScoringNote  `json:"scoring_notes"`
}

type HealthMetric struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Score    int    `json:"score"`
	MaxScore int    `json:"max_score"`
	Reason   string `json:"reason"`
}

type ScoringNote struct {
	Metric string `json:"metric"`
	Note   string `json:"note"`
}

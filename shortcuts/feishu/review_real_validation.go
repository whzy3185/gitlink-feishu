package feishu

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ReviewRealValidationSwitch = "FEISHU_REVIEW_REAL_VALIDATION"
	ReviewExternalWriteSwitch  = "FEISHU_REVIEW_CONFIRM_EXTERNAL_WRITES"
)

var (
	ErrReviewRealValidationDisabled = errors.New("real platform validation is disabled")
	ErrReviewExternalWritesDenied   = errors.New("real Feishu writes require explicit confirmation")
	reviewValidationRepository      = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)
	reviewValidationRunID           = regexp.MustCompile(`^review-final-[0-9]{8}T[0-9]{6}Z-[0-9a-f]{7,12}$`)
)

// ReviewRealValidationConfig is deliberately environment-only. Values that
// identify remote resources are never serialized by this type.
type ReviewRealValidationConfig struct {
	RunID             string
	GitLinkBaseURL    string
	GitLinkCredential string
	Repository        string
	PRNumber          int
	AppID             string
	AppSecret         string
	ChatA             string
	ChatB             string
	WebhookSecret     string
	WebhookURL        string
	BaseAppToken      string
	TableID           string
	DocumentFolder    string
	TaskContainer     string
	StateDatabase     string
	BackupDirectory   string
	ResourcePrefix    string
	AdminAddress      string
	WebhookAddress    string
	ExternalWrites    bool
	RealValidation    bool
}

func LoadReviewRealValidationConfigFromEnvironment() (ReviewRealValidationConfig, error) {
	prNumber, err := strconv.Atoi(strings.TrimSpace(os.Getenv("GITLINK_REVIEW_TEST_PR_NUMBER")))
	if err != nil {
		prNumber = 0
	}
	config := ReviewRealValidationConfig{
		RunID:             strings.TrimSpace(os.Getenv("FEISHU_REVIEW_VALIDATION_RUN_ID")),
		GitLinkBaseURL:    strings.TrimSpace(os.Getenv("GITLINK_REVIEW_BASE_URL")),
		GitLinkCredential: strings.TrimSpace(os.Getenv("GITLINK_REVIEW_CREDENTIAL_REF")),
		Repository:        strings.TrimSpace(os.Getenv("GITLINK_REVIEW_TEST_REPOSITORY")),
		PRNumber:          prNumber,
		AppID:             strings.TrimSpace(os.Getenv("FEISHU_APP_ID")),
		AppSecret:         strings.TrimSpace(os.Getenv("FEISHU_APP_SECRET")),
		ChatA:             strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_CHAT_A")),
		ChatB:             strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_CHAT_B")),
		WebhookSecret:     strings.TrimSpace(os.Getenv("FEISHU_REVIEW_WEBHOOK_SECRET")),
		WebhookURL:        strings.TrimSpace(os.Getenv("FEISHU_REVIEW_PUBLIC_WEBHOOK_URL")),
		BaseAppToken:      strings.TrimSpace(os.Getenv("FEISHU_REVIEW_BASE_APP_TOKEN")),
		TableID:           strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TABLE_ID")),
		DocumentFolder:    strings.TrimSpace(os.Getenv("FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN")),
		TaskContainer:     strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TASK_CONTAINER_ID")),
		StateDatabase:     strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_STATE_DB")),
		BackupDirectory:   strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_BACKUP_DIR")),
		ResourcePrefix:    strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_RESOURCE_PREFIX")),
		AdminAddress:      strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_ADMIN_ADDRESS")),
		WebhookAddress:    strings.TrimSpace(os.Getenv("FEISHU_REVIEW_TEST_WEBHOOK_ADDRESS")),
		RealValidation:    strings.TrimSpace(os.Getenv(ReviewRealValidationSwitch)) == "1",
		ExternalWrites:    strings.TrimSpace(os.Getenv(ReviewExternalWriteSwitch)) == "YES",
	}
	return config, nil
}

func (c ReviewRealValidationConfig) Validate(requireExternalWrites bool) error {
	if !c.RealValidation {
		return ErrReviewRealValidationDisabled
	}
	if requireExternalWrites && !c.ExternalWrites {
		return ErrReviewExternalWritesDenied
	}
	missing := []string{}
	for name, value := range map[string]string{
		"FEISHU_REVIEW_VALIDATION_RUN_ID":     c.RunID,
		"GITLINK_REVIEW_BASE_URL":             c.GitLinkBaseURL,
		"GITLINK_REVIEW_CREDENTIAL_REF":       c.GitLinkCredential,
		"GITLINK_REVIEW_TEST_REPOSITORY":      c.Repository,
		"FEISHU_APP_ID":                       c.AppID,
		"FEISHU_APP_SECRET":                   c.AppSecret,
		"FEISHU_REVIEW_TEST_CHAT_A":           c.ChatA,
		"FEISHU_REVIEW_TEST_CHAT_B":           c.ChatB,
		"FEISHU_REVIEW_WEBHOOK_SECRET":        c.WebhookSecret,
		"FEISHU_REVIEW_PUBLIC_WEBHOOK_URL":    c.WebhookURL,
		"FEISHU_REVIEW_BASE_APP_TOKEN":        c.BaseAppToken,
		"FEISHU_REVIEW_TABLE_ID":              c.TableID,
		"FEISHU_REVIEW_DOCUMENT_FOLDER_TOKEN": c.DocumentFolder,
		"FEISHU_REVIEW_TEST_STATE_DB":         c.StateDatabase,
		"FEISHU_REVIEW_TEST_BACKUP_DIR":       c.BackupDirectory,
		"FEISHU_REVIEW_TEST_RESOURCE_PREFIX":  c.ResourcePrefix,
		"FEISHU_REVIEW_TEST_ADMIN_ADDRESS":    c.AdminAddress,
		"FEISHU_REVIEW_TEST_WEBHOOK_ADDRESS":  c.WebhookAddress,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if c.PRNumber <= 0 {
		missing = append(missing, "GITLINK_REVIEW_TEST_PR_NUMBER")
	}
	if len(missing) > 0 {
		return fmt.Errorf("real validation configuration is incomplete: %s", strings.Join(missing, ", "))
	}
	if !reviewValidationRunID.MatchString(c.RunID) {
		return fmt.Errorf("validation run ID must use review-final-<UTC>-<short-sha>")
	}
	if !reviewValidationRepository.MatchString(c.Repository) {
		return fmt.Errorf("GitLink test repository must use owner/repo")
	}
	if c.ChatA == c.ChatB {
		return fmt.Errorf("real validation requires two different Feishu test chats")
	}
	if !strings.HasPrefix(c.ResourcePrefix, "review-final-") {
		return fmt.Errorf("test resource prefix must start with review-final-")
	}
	if err := validateReviewValidationHTTPS(c.GitLinkBaseURL, "GitLink base URL"); err != nil {
		return err
	}
	if err := validateReviewValidationHTTPS(c.WebhookURL, "public webhook URL"); err != nil {
		return err
	}
	if !strings.HasSuffix(strings.ToLower(c.StateDatabase), ".db") || !strings.Contains(strings.ToLower(filepath.Base(c.StateDatabase)), "test") {
		return fmt.Errorf("state database must be a dedicated test .db file")
	}
	if !strings.Contains(strings.ToLower(filepath.Base(c.BackupDirectory)), "test") {
		return fmt.Errorf("backup directory must be dedicated to tests")
	}
	if c.AdminAddress == c.WebhookAddress {
		return fmt.Errorf("admin and webhook listeners must use different addresses")
	}
	for label, address := range map[string]string{"admin": c.AdminAddress, "webhook": c.WebhookAddress} {
		if _, _, err := net.SplitHostPort(address); err != nil {
			return fmt.Errorf("%s listener address is invalid: %w", label, err)
		}
	}
	if !strings.HasPrefix(c.GitLinkCredential, "env:") || strings.TrimSpace(os.Getenv(strings.TrimPrefix(c.GitLinkCredential, "env:"))) == "" {
		return fmt.Errorf("GitLink credential reference must resolve from env:VARIABLE")
	}
	return nil
}

func NewReviewValidationRunID(now time.Time, shortSHA string) (string, error) {
	shortSHA = strings.ToLower(strings.TrimSpace(shortSHA))
	if matched, _ := regexp.MatchString(`^[0-9a-f]{7,12}$`, shortSHA); !matched {
		return "", fmt.Errorf("short SHA must contain 7 to 12 hexadecimal characters")
	}
	return fmt.Sprintf("review-final-%s-%s", now.UTC().Format("20060102T150405Z"), shortSHA), nil
}

func HashReviewValidationIdentifier(value string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(digest[:8])
}

func validateReviewValidationHTTPS(raw, label string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute HTTPS URL", label)
	}
	return nil
}

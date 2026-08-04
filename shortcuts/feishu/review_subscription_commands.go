package feishu

import (
	"regexp"
	"strings"
)

var (
	reviewSubscribePattern   = regexp.MustCompile(`(?i)^(?:订阅|subscribe)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+(.+)$`)
	reviewUnsubscribePattern = regexp.MustCompile(`(?i)^(?:取消订阅|unsubscribe)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(?:\s+(.+))?$`)
	reviewShowSubscriptions  = regexp.MustCompile(`(?i)^(?:查看订阅|show subscriptions?)(?:\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+))?$`)
	reviewNotificationMode   = regexp.MustCompile(`(?i)^(?:设置通知|notification mode)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)\s+(canonical_only|canonical_and_notice|silent_refresh)$`)
	reviewDefaultRepository  = regexp.MustCompile(`(?i)^(?:设置默认仓库|set default repository)\s+([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)$`)
)

func parseReviewSubscriptionIntent(content string) (ReviewGatewayIntent, bool) {
	if match := reviewSubscribePattern.FindStringSubmatch(content); len(match) == 3 {
		groups := strings.Join(strings.Fields(match[2]), ",")
		return ReviewGatewayIntent{Name: "subscribe_review_events", Repository: match[1], Argument: groups}, true
	}
	if match := reviewUnsubscribePattern.FindStringSubmatch(content); len(match) == 3 {
		groups := strings.Join(strings.Fields(match[2]), ",")
		if groups == "" {
			groups = strings.Join([]string{
				ReviewSubscriptionGroupPulls, ReviewSubscriptionGroupReviews,
				ReviewSubscriptionGroupThreads, ReviewSubscriptionGroupMerge,
				ReviewSubscriptionGroupCI,
			}, ",")
		}
		return ReviewGatewayIntent{Name: "unsubscribe_review_events", Repository: match[1], Argument: groups}, true
	}
	if match := reviewShowSubscriptions.FindStringSubmatch(content); len(match) == 2 {
		return ReviewGatewayIntent{Name: "show_review_subscriptions", Repository: match[1]}, true
	}
	if match := reviewNotificationMode.FindStringSubmatch(content); len(match) == 3 {
		return ReviewGatewayIntent{Name: "set_review_notification_mode", Repository: match[1], Argument: strings.ToLower(match[2])}, true
	}
	if match := reviewDefaultRepository.FindStringSubmatch(content); len(match) == 2 {
		return ReviewGatewayIntent{Name: "set_default_review_repository", Repository: match[1]}, true
	}
	return ReviewGatewayIntent{}, false
}

func splitReviewSubscriptionArgument(argument string) []string {
	if strings.TrimSpace(argument) == "" {
		return nil
	}
	return strings.FieldsFunc(argument, func(value rune) bool {
		return value == ',' || value == '，' || value == ' '
	})
}

func reviewGatewayIntentRequiresAdmin(action string) bool {
	switch action {
	case "plan_bind_repository", "subscribe_review_events", "unsubscribe_review_events",
		"set_review_notification_mode", "set_default_review_repository":
		return true
	default:
		return false
	}
}

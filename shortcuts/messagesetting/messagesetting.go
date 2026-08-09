package messagesetting

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/gitlink-org/gitlink-cli/internal/i18n"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

const (
	channelNotification = "notification"
	channelEmail        = "email"
	channelBoth         = "both"
)

type catalogResponse struct {
	SettingTypes []catalogGroup `json:"setting_types"`
}

type catalogGroup struct {
	Type     string              `json:"type"`
	TypeName string              `json:"type_name"`
	Settings []catalogSettingRow `json:"settings"`
}

type catalogSettingRow struct {
	Name                 string `json:"name"`
	Key                  string `json:"key"`
	NotificationDisabled bool   `json:"notification_disabled"`
	EmailDisabled        bool   `json:"email_disabled"`
}

type userSettingResponse struct {
	User             map[string]interface{} `json:"user"`
	NotificationBody map[string]bool        `json:"notification_body"`
	EmailBody        map[string]bool        `json:"email_body"`
}

type currentUserResponse struct {
	Login string `json:"login"`
}

type settingMeta struct {
	FullKey                    string
	ShortKey                   string
	Group                      string
	GroupName                  string
	Name                       string
	DefaultNotificationEnabled *bool
	DefaultEmailEnabled        *bool
}

type settingRegistry struct {
	OrderedGroups []string
	GroupNames    map[string]string
	GroupOrder    map[string]int
	Items         map[string]settingMeta
}

type groupedSettingOutput struct {
	Group     string               `json:"group"`
	GroupName string               `json:"group_name,omitempty"`
	Settings  []settingOutput      `json:"settings"`
	Summary   channelSummaryOutput `json:"summary"`
}

type settingOutput struct {
	Key                        string `json:"key"`
	ShortKey                   string `json:"short_key"`
	Name                       string `json:"name"`
	NotificationEnabled        bool   `json:"notification_enabled"`
	EmailEnabled               bool   `json:"email_enabled"`
	DefaultNotificationEnabled *bool  `json:"default_notification_enabled,omitempty"`
	DefaultEmailEnabled        *bool  `json:"default_email_enabled,omitempty"`
}

type channelSummaryOutput struct {
	Total               int `json:"total"`
	NotificationEnabled int `json:"notification_enabled"`
	EmailEnabled        int `json:"email_enabled"`
}

type settingChangeOutput struct {
	Key                string `json:"key"`
	ShortKey           string `json:"short_key"`
	Name               string `json:"name"`
	Group              string `json:"group"`
	GroupName          string `json:"group_name,omitempty"`
	NotificationBefore bool   `json:"notification_before"`
	NotificationAfter  bool   `json:"notification_after"`
	EmailBefore        bool   `json:"email_before"`
	EmailAfter         bool   `json:"email_after"`
}

type presetSpec struct {
	Notification bool
	Email        bool
}

var allowedPresetNames = map[string]presetSpec{
	"all-on":            {Notification: true, Email: true},
	"all-off":           {Notification: false, Email: false},
	"notification-only": {Notification: true, Email: false},
	"email-only":        {Notification: false, Email: true},
}

// Shortcuts returns message settings shortcuts.
func Shortcuts(translators ...*i18n.Translator) []*common.Shortcut {
	tr := shortcutTranslator(translators...)
	return []*common.Shortcut{
		{
			Name:        "catalog",
			Description: tr.T("cmd.message_settings.catalog.short"),
			Flags: []common.Flag{
				{Name: "group", Usage: tr.T("flag.message_settings.group")},
			},
			Run: runCatalog,
		},
		{
			Name:        "view",
			Description: tr.T("cmd.message_settings.view.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.message_settings.login")},
				{Name: "group", Usage: tr.T("flag.message_settings.group")},
			},
			Run: runView,
		},
		{
			Name:        "update",
			Description: tr.T("cmd.message_settings.update.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.message_settings.login")},
				{Name: "channel", Usage: tr.T("flag.message_settings.channel"), Required: true},
				{Name: "state", Usage: tr.T("flag.message_settings.state"), Required: true},
				{Name: "keys", Usage: tr.T("flag.message_settings.keys")},
				{Name: "group", Usage: tr.T("flag.message_settings.group")},
				{Name: "all", Usage: tr.T("flag.message_settings.all"), Bool: true, Default: "false"},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runUpdate,
		},
		{
			Name:        "preset",
			Description: tr.T("cmd.message_settings.preset.short"),
			Flags: []common.Flag{
				{Name: "login", Short: "l", Usage: tr.T("flag.message_settings.login")},
				{Name: "name", Usage: tr.T("flag.message_settings.preset_name"), Required: true},
				{Name: "keys", Usage: tr.T("flag.message_settings.keys")},
				{Name: "group", Usage: tr.T("flag.message_settings.group")},
				{Name: "all", Usage: tr.T("flag.message_settings.all"), Bool: true, Default: "false"},
				{Name: "dry-run", Usage: tr.T("flag.dry_run"), Bool: true, Default: "false"},
			},
			Run: runPreset,
		},
	}
}

func shortcutTranslator(translators ...*i18n.Translator) *i18n.Translator {
	if len(translators) > 0 && translators[0] != nil {
		return translators[0]
	}
	return i18n.Default()
}

func runCatalog(ctx *common.RuntimeContext) error {
	registry, err := loadRegistry(ctx)
	if err != nil {
		return err
	}

	groups, err := parseGroupFilter(ctx.Arg("group"), registry)
	if err != nil {
		return err
	}

	outputGroups := renderCatalogGroups(registry, groups)
	return ctx.OutputData(map[string]interface{}{
		"action":  "catalog_message_settings",
		"groups":  outputGroups,
		"summary": summarizeCatalogGroups(outputGroups),
	})
}

func runView(ctx *common.RuntimeContext) error {
	login, registry, current, err := loadUserSettingContext(ctx)
	if err != nil {
		return err
	}

	groups, err := parseGroupFilter(ctx.Arg("group"), registry)
	if err != nil {
		return err
	}

	outputGroups := renderUserGroups(registry, current, groups)
	return ctx.OutputData(map[string]interface{}{
		"action":  "view_message_settings",
		"login":   login,
		"user":    current.User,
		"groups":  outputGroups,
		"summary": summarizeUserGroups(outputGroups),
	})
}

func runUpdate(ctx *common.RuntimeContext) error {
	channel, err := parseChannel(ctx.Arg("channel"))
	if err != nil {
		return err
	}
	state, err := parseState(ctx.Arg("state"))
	if err != nil {
		return err
	}

	result, err := updateSettings(ctx, updateRequest{
		Action:   "update_message_settings",
		Channel:  channel,
		Selector: selectorArgsFromContext(ctx),
		DryRun:   parseBoolArg(ctx.Arg("dry-run")),
		Apply: func(notificationCurrent, emailCurrent bool) (bool, bool) {
			nextNotification := notificationCurrent
			nextEmail := emailCurrent
			switch channel {
			case channelNotification:
				nextNotification = state
			case channelEmail:
				nextEmail = state
			case channelBoth:
				nextNotification = state
				nextEmail = state
			}
			return nextNotification, nextEmail
		},
		Extra: map[string]interface{}{
			"channel": channel,
			"state":   state,
		},
	})
	if err != nil {
		return err
	}
	return ctx.OutputData(result)
}

func runPreset(ctx *common.RuntimeContext) error {
	name := strings.ToLower(strings.TrimSpace(ctx.Arg("name")))
	preset, ok := allowedPresetNames[name]
	if !ok {
		return fmt.Errorf("invalid --name value %q", ctx.Arg("name"))
	}

	result, err := updateSettings(ctx, updateRequest{
		Action:   "preset_message_settings",
		Selector: selectorArgsFromContext(ctx),
		DryRun:   parseBoolArg(ctx.Arg("dry-run")),
		Apply: func(_, _ bool) (bool, bool) {
			return preset.Notification, preset.Email
		},
		Extra: map[string]interface{}{
			"preset": name,
		},
	})
	if err != nil {
		return err
	}
	return ctx.OutputData(result)
}

type selectorArgs struct {
	Keys  string
	Group string
	All   bool
}

type updateRequest struct {
	Action   string
	Channel  string
	Selector selectorArgs
	DryRun   bool
	Apply    func(notificationCurrent, emailCurrent bool) (bool, bool)
	Extra    map[string]interface{}
}

func selectorArgsFromContext(ctx *common.RuntimeContext) selectorArgs {
	return selectorArgs{
		Keys:  ctx.Arg("keys"),
		Group: ctx.Arg("group"),
		All:   parseBoolArg(ctx.Arg("all")),
	}
}

func updateSettings(ctx *common.RuntimeContext, req updateRequest) (map[string]interface{}, error) {
	login, registry, current, err := loadUserSettingContext(ctx)
	if err != nil {
		return nil, err
	}

	selectedKeys, selectedGroups, err := selectKeys(req.Selector, registry)
	if err != nil {
		return nil, err
	}

	notificationBody, emailBody := mergedBodies(current, registry)
	changes := make([]settingChangeOutput, 0, len(selectedKeys))
	changedKeys := make([]string, 0, len(selectedKeys))
	for _, key := range selectedKeys {
		meta := registry.Items[key]
		notificationBefore := notificationBody[key]
		emailBefore := emailBody[key]
		notificationAfter, emailAfter := req.Apply(notificationBefore, emailBefore)
		notificationBody[key] = notificationAfter
		emailBody[key] = emailAfter

		change := settingChangeOutput{
			Key:                key,
			ShortKey:           meta.ShortKey,
			Name:               meta.Name,
			Group:              meta.Group,
			GroupName:          meta.GroupName,
			NotificationBefore: notificationBefore,
			NotificationAfter:  notificationAfter,
			EmailBefore:        emailBefore,
			EmailAfter:         emailAfter,
		}
		changes = append(changes, change)
		if notificationBefore != notificationAfter || emailBefore != emailAfter {
			changedKeys = append(changedKeys, key)
		}
	}

	result := map[string]interface{}{
		"action":          req.Action,
		"login":           login,
		"user":            current.User,
		"dry_run":         req.DryRun,
		"selected_keys":   selectedKeys,
		"selected_groups": selectedGroups,
		"changed_keys":    changedKeys,
		"changes":         changes,
		"summary": map[string]interface{}{
			"selected": len(selectedKeys),
			"changed":  len(changedKeys),
		},
		"setting": map[string]interface{}{
			"notification_body": notificationBody,
			"email_body":        emailBody,
		},
	}
	for key, value := range req.Extra {
		result[key] = value
	}

	if req.DryRun {
		return result, nil
	}

	payload := map[string]interface{}{
		"setting": map[string]interface{}{
			"notification_body": notificationBody,
			"email_body":        emailBody,
		},
	}
	env, err := ctx.CallAPI("POST", fmt.Sprintf("/api/users/%s/template_message_settings/update_setting", login), payload)
	if err != nil {
		return nil, err
	}
	result["updated"] = env.Data
	return result, nil
}

func loadRegistry(ctx *common.RuntimeContext) (*settingRegistry, error) {
	catalog, err := fetchCatalog(ctx)
	if err != nil {
		return nil, err
	}
	return buildRegistry(catalog, nil), nil
}

func loadUserSettingContext(ctx *common.RuntimeContext) (string, *settingRegistry, *userSettingResponse, error) {
	login, err := resolveTargetLogin(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	catalog, err := fetchCatalog(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	current, err := fetchUserSettings(ctx, login)
	if err != nil {
		return "", nil, nil, err
	}
	return login, buildRegistry(catalog, current), current, nil
}

func resolveTargetLogin(ctx *common.RuntimeContext) (string, error) {
	if login := strings.TrimSpace(ctx.Arg("login")); login != "" {
		return login, nil
	}

	env, err := ctx.CallAPI("GET", "/users/me", nil)
	if err != nil {
		return "", fmt.Errorf("fetch current user: %w", err)
	}
	var current currentUserResponse
	if err := decodeEnvelopeData(env.Data, &current); err != nil {
		return "", fmt.Errorf("parse current user: %w", err)
	}
	if strings.TrimSpace(current.Login) == "" {
		return "", fmt.Errorf("current user response did not include a login")
	}
	return current.Login, nil
}

func fetchCatalog(ctx *common.RuntimeContext) (*catalogResponse, error) {
	env, err := ctx.CallAPI("GET", "/api/template_message_settings", nil)
	if err != nil {
		return nil, fmt.Errorf("fetch message setting catalog: %w", err)
	}
	var catalog catalogResponse
	if err := decodeEnvelopeData(env.Data, &catalog); err != nil {
		return nil, fmt.Errorf("parse message setting catalog: %w", err)
	}
	return &catalog, nil
}

func fetchUserSettings(ctx *common.RuntimeContext, login string) (*userSettingResponse, error) {
	env, err := ctx.CallAPI("GET", fmt.Sprintf("/api/users/%s/template_message_settings", login), nil)
	if err != nil {
		return nil, fmt.Errorf("fetch user message settings: %w", err)
	}
	var response userSettingResponse
	if err := decodeEnvelopeData(env.Data, &response); err != nil {
		return nil, fmt.Errorf("parse user message settings: %w", err)
	}
	if response.NotificationBody == nil {
		response.NotificationBody = map[string]bool{}
	}
	if response.EmailBody == nil {
		response.EmailBody = map[string]bool{}
	}
	return &response, nil
}

func buildRegistry(catalog *catalogResponse, current *userSettingResponse) *settingRegistry {
	registry := &settingRegistry{
		OrderedGroups: []string{},
		GroupNames:    map[string]string{},
		GroupOrder:    map[string]int{},
		Items:         map[string]settingMeta{},
	}

	if catalog != nil {
		for _, group := range catalog.SettingTypes {
			shortGroup := shortenSettingType(group.Type)
			if shortGroup == "" {
				continue
			}
			ensureGroup(registry, shortGroup, group.TypeName)
			for _, row := range group.Settings {
				fullKey := shortGroup + "::" + strings.TrimSpace(row.Key)
				defaultNotification := !row.NotificationDisabled
				defaultEmail := !row.EmailDisabled
				registry.Items[fullKey] = settingMeta{
					FullKey:                    fullKey,
					ShortKey:                   strings.TrimSpace(row.Key),
					Group:                      shortGroup,
					GroupName:                  group.TypeName,
					Name:                       strings.TrimSpace(row.Name),
					DefaultNotificationEnabled: boolPtr(defaultNotification),
					DefaultEmailEnabled:        boolPtr(defaultEmail),
				}
			}
		}
	}

	if current != nil {
		for key := range current.NotificationBody {
			ensureSettingMeta(registry, key)
		}
		for key := range current.EmailBody {
			ensureSettingMeta(registry, key)
		}
	}

	return registry
}

func ensureGroup(registry *settingRegistry, group, name string) {
	group = strings.TrimSpace(group)
	if group == "" {
		return
	}
	if _, ok := registry.GroupOrder[group]; !ok {
		registry.GroupOrder[group] = len(registry.OrderedGroups)
		registry.OrderedGroups = append(registry.OrderedGroups, group)
	}
	if strings.TrimSpace(name) != "" {
		registry.GroupNames[group] = strings.TrimSpace(name)
	}
}

func ensureSettingMeta(registry *settingRegistry, fullKey string) {
	if _, ok := registry.Items[fullKey]; ok {
		return
	}
	group, shortKey := splitSettingKey(fullKey)
	ensureGroup(registry, group, registry.GroupNames[group])
	name := shortKey
	if name == "" {
		name = fullKey
	}
	registry.Items[fullKey] = settingMeta{
		FullKey:   fullKey,
		ShortKey:  shortKey,
		Group:     group,
		GroupName: registry.GroupNames[group],
		Name:      name,
	}
}

func renderCatalogGroups(registry *settingRegistry, filter []string) []groupedSettingOutput {
	allowedGroups := makeGroupSet(filter)
	groups := make([]groupedSettingOutput, 0, len(registry.OrderedGroups))
	for _, group := range orderedGroups(registry, filter) {
		if len(allowedGroups) > 0 && !allowedGroups[group] {
			continue
		}
		settings := collectSettings(registry, group)
		items := make([]settingOutput, 0, len(settings))
		summary := channelSummaryOutput{Total: len(settings)}
		for _, meta := range settings {
			item := settingOutput{
				Key:                        meta.FullKey,
				ShortKey:                   meta.ShortKey,
				Name:                       meta.Name,
				DefaultNotificationEnabled: meta.DefaultNotificationEnabled,
				DefaultEmailEnabled:        meta.DefaultEmailEnabled,
			}
			if meta.DefaultNotificationEnabled != nil && *meta.DefaultNotificationEnabled {
				item.NotificationEnabled = true
				summary.NotificationEnabled++
			}
			if meta.DefaultEmailEnabled != nil && *meta.DefaultEmailEnabled {
				item.EmailEnabled = true
				summary.EmailEnabled++
			}
			items = append(items, item)
		}
		groups = append(groups, groupedSettingOutput{
			Group:     group,
			GroupName: registry.GroupNames[group],
			Settings:  items,
			Summary:   summary,
		})
	}
	return groups
}

func renderUserGroups(registry *settingRegistry, current *userSettingResponse, filter []string) []groupedSettingOutput {
	notificationBody, emailBody := mergedBodies(current, registry)
	allowedGroups := makeGroupSet(filter)
	groups := make([]groupedSettingOutput, 0, len(registry.OrderedGroups))
	for _, group := range orderedGroups(registry, filter) {
		if len(allowedGroups) > 0 && !allowedGroups[group] {
			continue
		}
		settings := collectSettings(registry, group)
		items := make([]settingOutput, 0, len(settings))
		summary := channelSummaryOutput{Total: len(settings)}
		for _, meta := range settings {
			notificationEnabled := notificationBody[meta.FullKey]
			emailEnabled := emailBody[meta.FullKey]
			if notificationEnabled {
				summary.NotificationEnabled++
			}
			if emailEnabled {
				summary.EmailEnabled++
			}
			items = append(items, settingOutput{
				Key:                        meta.FullKey,
				ShortKey:                   meta.ShortKey,
				Name:                       meta.Name,
				NotificationEnabled:        notificationEnabled,
				EmailEnabled:               emailEnabled,
				DefaultNotificationEnabled: meta.DefaultNotificationEnabled,
				DefaultEmailEnabled:        meta.DefaultEmailEnabled,
			})
		}
		groups = append(groups, groupedSettingOutput{
			Group:     group,
			GroupName: registry.GroupNames[group],
			Settings:  items,
			Summary:   summary,
		})
	}
	return groups
}

func summarizeCatalogGroups(groups []groupedSettingOutput) channelSummaryOutput {
	summary := channelSummaryOutput{}
	for _, group := range groups {
		summary.Total += group.Summary.Total
		summary.NotificationEnabled += group.Summary.NotificationEnabled
		summary.EmailEnabled += group.Summary.EmailEnabled
	}
	return summary
}

func summarizeUserGroups(groups []groupedSettingOutput) channelSummaryOutput {
	return summarizeCatalogGroups(groups)
}

func selectKeys(args selectorArgs, registry *settingRegistry) ([]string, []string, error) {
	selected := map[string]bool{}

	groupFilter, err := parseGroupFilter(args.Group, registry)
	if err != nil {
		return nil, nil, err
	}
	for _, group := range groupFilter {
		for _, meta := range collectSettings(registry, group) {
			selected[meta.FullKey] = true
		}
	}

	keys, err := resolveKeyFilter(args.Keys, registry)
	if err != nil {
		return nil, nil, err
	}
	for _, key := range keys {
		selected[key] = true
	}

	if args.All {
		for key := range registry.Items {
			selected[key] = true
		}
	}

	if len(selected) == 0 {
		return nil, nil, fmt.Errorf("one of --keys, --group, or --all is required")
	}

	selectedKeys := make([]string, 0, len(selected))
	for key := range selected {
		selectedKeys = append(selectedKeys, key)
	}
	sortKeys(registry, selectedKeys)

	selectedGroupsSet := map[string]bool{}
	for _, key := range selectedKeys {
		selectedGroupsSet[registry.Items[key].Group] = true
	}
	selectedGroups := make([]string, 0, len(selectedGroupsSet))
	for _, group := range registry.OrderedGroups {
		if selectedGroupsSet[group] {
			selectedGroups = append(selectedGroups, group)
		}
	}
	return selectedKeys, selectedGroups, nil
}

func parseGroupFilter(value string, registry *settingRegistry) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	groupLookup := map[string]string{}
	for _, group := range registry.OrderedGroups {
		groupLookup[strings.ToLower(group)] = group
	}

	parts := strings.Split(value, ",")
	groups := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		token := strings.ToLower(strings.TrimSpace(part))
		if token == "" {
			continue
		}
		group, ok := groupLookup[token]
		if !ok {
			return nil, fmt.Errorf("unknown --group value %q", strings.TrimSpace(part))
		}
		if seen[group] {
			continue
		}
		seen[group] = true
		groups = append(groups, group)
	}
	return groups, nil
}

func resolveKeyFilter(value string, registry *settingRegistry) ([]string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	shortLookup := map[string][]string{}
	fullLookup := map[string]string{}
	for key, meta := range registry.Items {
		shortLookup[strings.ToLower(meta.ShortKey)] = append(shortLookup[strings.ToLower(meta.ShortKey)], key)
		fullLookup[strings.ToLower(meta.FullKey)] = key
	}

	parts := strings.Split(value, ",")
	keys := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			continue
		}

		var resolved string
		if strings.Contains(token, "::") {
			var ok bool
			resolved, ok = fullLookup[strings.ToLower(token)]
			if !ok {
				return nil, fmt.Errorf("unknown --keys value %q", token)
			}
		} else {
			matches := shortLookup[strings.ToLower(token)]
			switch len(matches) {
			case 0:
				return nil, fmt.Errorf("unknown --keys value %q", token)
			case 1:
				resolved = matches[0]
			default:
				return nil, fmt.Errorf("ambiguous --keys value %q; use a full Group::Key value", token)
			}
		}

		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		keys = append(keys, resolved)
	}
	sortKeys(registry, keys)
	return keys, nil
}

func mergedBodies(current *userSettingResponse, registry *settingRegistry) (map[string]bool, map[string]bool) {
	notificationBody := map[string]bool{}
	emailBody := map[string]bool{}
	for key, value := range current.NotificationBody {
		notificationBody[key] = value
	}
	for key, value := range current.EmailBody {
		emailBody[key] = value
	}
	for key, meta := range registry.Items {
		if _, ok := notificationBody[key]; !ok {
			if meta.DefaultNotificationEnabled != nil {
				notificationBody[key] = *meta.DefaultNotificationEnabled
			} else {
				notificationBody[key] = false
			}
		}
		if _, ok := emailBody[key]; !ok {
			if meta.DefaultEmailEnabled != nil {
				emailBody[key] = *meta.DefaultEmailEnabled
			} else {
				emailBody[key] = false
			}
		}
	}
	return notificationBody, emailBody
}

func collectSettings(registry *settingRegistry, group string) []settingMeta {
	settings := []settingMeta{}
	for _, meta := range registry.Items {
		if meta.Group != group {
			continue
		}
		settings = append(settings, meta)
	}
	sort.Slice(settings, func(i, j int) bool {
		return strings.ToLower(settings[i].ShortKey) < strings.ToLower(settings[j].ShortKey)
	})
	return settings
}

func orderedGroups(registry *settingRegistry, filter []string) []string {
	if len(filter) == 0 {
		return append([]string(nil), registry.OrderedGroups...)
	}
	return append([]string(nil), filter...)
}

func sortKeys(registry *settingRegistry, keys []string) {
	sort.Slice(keys, func(i, j int) bool {
		left := registry.Items[keys[i]]
		right := registry.Items[keys[j]]
		leftOrder := registry.GroupOrder[left.Group]
		rightOrder := registry.GroupOrder[right.Group]
		if leftOrder != rightOrder {
			return leftOrder < rightOrder
		}
		return strings.ToLower(left.ShortKey) < strings.ToLower(right.ShortKey)
	})
}

func parseChannel(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case channelNotification:
		return channelNotification, nil
	case channelEmail:
		return channelEmail, nil
	case channelBoth:
		return channelBoth, nil
	default:
		return "", fmt.Errorf("invalid --channel value %q", value)
	}
}

func parseState(value string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "on", "true", "enable", "enabled":
		return true, nil
	case "off", "false", "disable", "disabled":
		return false, nil
	default:
		return false, fmt.Errorf("invalid --state value %q", value)
	}
}

func parseBoolArg(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func splitSettingKey(fullKey string) (string, string) {
	parts := strings.SplitN(strings.TrimSpace(fullKey), "::", 2)
	if len(parts) != 2 {
		return strings.TrimSpace(fullKey), strings.TrimSpace(fullKey)
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
}

func shortenSettingType(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	parts := strings.Split(value, "::")
	return strings.TrimSpace(parts[len(parts)-1])
}

func makeGroupSet(groups []string) map[string]bool {
	if len(groups) == 0 {
		return nil
	}
	result := map[string]bool{}
	for _, group := range groups {
		result[group] = true
	}
	return result
}

func boolPtr(value bool) *bool {
	return &value
}

func decodeEnvelopeData(data interface{}, target interface{}) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

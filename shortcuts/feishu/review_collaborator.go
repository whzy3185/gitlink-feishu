package feishu

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type ReviewRepositoryCollaborator struct {
	Login string `json:"login"`
	Name  string `json:"name,omitempty"`
}

type ReviewRepositoryCollaboratorReader interface {
	ListRepositoryCollaborators(context.Context, *common.RuntimeContext, string, string) ([]ReviewRepositoryCollaborator, error)
}

type GitLinkReviewRepositoryCollaboratorReader struct{}

func (GitLinkReviewRepositoryCollaboratorReader) ListRepositoryCollaborators(
	ctx context.Context,
	runtime *common.RuntimeContext,
	owner,
	repository string,
) ([]ReviewRepositoryCollaborator, error) {
	if runtime == nil || runtime.Client == nil {
		return nil, fmt.Errorf("GitLink runtime is required")
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	envelope, err := runtime.Client.Get(fmt.Sprintf("/v1/%s/%s/collaborators", owner, repository), nil)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(envelope.Data)
	if err != nil {
		return nil, fmt.Errorf("encode GitLink repository collaborators: %w", err)
	}
	var response map[string]interface{}
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("decode GitLink repository collaborators: %w", err)
	}
	items := response["data"]
	if items == nil {
		items = response["collaborators"]
	}
	if items == nil {
		items = response["members"]
	}
	itemData, err := json.Marshal(items)
	if err != nil {
		return nil, fmt.Errorf("encode GitLink collaborator list: %w", err)
	}
	var collaborators []ReviewRepositoryCollaborator
	if err := json.Unmarshal(itemData, &collaborators); err != nil {
		return nil, fmt.Errorf("decode GitLink collaborator list: %w", err)
	}
	for index := range collaborators {
		collaborators[index].Login = strings.TrimSpace(collaborators[index].Login)
		collaborators[index].Name = strings.TrimSpace(collaborators[index].Name)
	}
	return collaborators, nil
}

type FeishuDisplayNameResolver interface {
	ResolveFeishuDisplayName(context.Context, string) (string, error)
}

type reviewDisplayNameCacheEntry struct {
	name      string
	expiresAt time.Time
}

type ReviewFeishuDisplayNameResolver struct {
	Client        OpenAPIClient
	TokenProvider *ReviewTenantTokenProvider
	AppID         string
	AppSecret     string
	Now           func() time.Time

	mu    sync.Mutex
	cache map[string]reviewDisplayNameCacheEntry
}

func NewReviewFeishuDisplayNameResolver(
	client OpenAPIClient,
	tokens *ReviewTenantTokenProvider,
	appID,
	appSecret string,
) *ReviewFeishuDisplayNameResolver {
	return &ReviewFeishuDisplayNameResolver{
		Client: client, TokenProvider: tokens, AppID: appID, AppSecret: appSecret,
		Now: time.Now, cache: map[string]reviewDisplayNameCacheEntry{},
	}
}

func (r *ReviewFeishuDisplayNameResolver) ResolveFeishuDisplayName(ctx context.Context, userID string) (string, error) {
	userID = strings.TrimSpace(userID)
	if r == nil || userID == "" {
		return "", fmt.Errorf("Feishu user ID is required")
	}
	now := time.Now()
	if r.Now != nil {
		now = r.Now()
	}
	r.mu.Lock()
	if entry, ok := r.cache[userID]; ok && entry.expiresAt.After(now) {
		r.mu.Unlock()
		return entry.name, nil
	}
	r.mu.Unlock()
	token, err := r.TokenProvider.Token(ctx, r.AppID, r.AppSecret)
	if err != nil {
		return "", err
	}
	name, err := r.Client.GetUserDisplayName(ctx, token, userID)
	if err != nil {
		return "", err
	}
	r.mu.Lock()
	r.cache[userID] = reviewDisplayNameCacheEntry{name: name, expiresAt: now.Add(10 * time.Minute)}
	r.mu.Unlock()
	return name, nil
}

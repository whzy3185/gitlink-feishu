package feishu

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

type reviewTenantTokenClient interface {
	TenantAccessToken(context.Context, string, string) (TenantToken, error)
}

type reviewTenantTokenEntry struct {
	token     TenantToken
	expiresAt time.Time
}

type reviewTenantTokenFlight struct {
	done chan struct{}
	err  error
}

type ReviewTenantTokenProvider struct {
	client reviewTenantTokenClient
	now    func() time.Time

	mu      sync.Mutex
	entries map[string]reviewTenantTokenEntry
	flights map[string]*reviewTenantTokenFlight
}

func NewReviewTenantTokenProvider(client reviewTenantTokenClient, now func() time.Time) *ReviewTenantTokenProvider {
	if now == nil {
		now = time.Now
	}
	return &ReviewTenantTokenProvider{
		client:  client,
		now:     now,
		entries: map[string]reviewTenantTokenEntry{},
		flights: map[string]*reviewTenantTokenFlight{},
	}
}

func (p *ReviewTenantTokenProvider) Token(ctx context.Context, appID, appSecret string) (string, error) {
	appID = strings.TrimSpace(appID)
	if p == nil || p.client == nil || appID == "" || strings.TrimSpace(appSecret) == "" {
		return "", fmt.Errorf("Feishu app credentials are required")
	}
	for {
		p.mu.Lock()
		if entry, ok := p.entries[appID]; ok && entry.expiresAt.After(p.now().UTC().Add(2*time.Minute)) {
			p.mu.Unlock()
			return entry.token.Value, nil
		}
		if flight, ok := p.flights[appID]; ok {
			p.mu.Unlock()
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-flight.done:
				if flight.err != nil {
					return "", flight.err
				}
				continue
			}
		}
		flight := &reviewTenantTokenFlight{done: make(chan struct{})}
		p.flights[appID] = flight
		p.mu.Unlock()

		token, err := p.client.TenantAccessToken(ctx, appID, appSecret)
		p.mu.Lock()
		if err == nil && strings.TrimSpace(token.Value) != "" {
			expires := token.Expire
			if expires <= 0 {
				expires = 7200
			}
			p.entries[appID] = reviewTenantTokenEntry{token: token, expiresAt: p.now().UTC().Add(time.Duration(expires) * time.Second)}
		} else if err == nil {
			err = fmt.Errorf("Feishu tenant token response missing value")
		}
		flight.err = err
		delete(p.flights, appID)
		close(flight.done)
		p.mu.Unlock()
		if err != nil {
			return "", err
		}
		return token.Value, nil
	}
}

func (p *ReviewTenantTokenProvider) Invalidate(appID string) {
	if p == nil {
		return
	}
	p.mu.Lock()
	delete(p.entries, strings.TrimSpace(appID))
	p.mu.Unlock()
}

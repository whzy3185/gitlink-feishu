package feishu

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	internalAuth "github.com/gitlink-org/gitlink-cli/internal/auth"
	"github.com/gitlink-org/gitlink-cli/shortcuts/common"
)

type reviewGatewayStaticAuthTransport struct {
	base       http.RoundTripper
	credential string
}

func (t *reviewGatewayStaticAuthTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	clone := request.Clone(request.Context())
	clone.URL = cloneURL(request.URL)
	credential := t.credential
	if strings.HasPrefix(credential, "cookie:") {
		cookie := strings.TrimPrefix(credential, "cookie:")
		if existing := clone.Header.Get("Cookie"); existing != "" {
			clone.Header.Set("Cookie", existing+"; "+cookie)
		} else {
			clone.Header.Set("Cookie", cookie)
		}
	} else if credential != "" {
		query := clone.URL.Query()
		query.Set("access_token", credential)
		clone.URL.RawQuery = query.Encode()
	}
	if clone.Body != nil && clone.Header.Get("Content-Type") == "" {
		clone.Header.Set("Content-Type", "application/json")
	}
	clone.Header.Set("Accept", "application/json")
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

func cloneURL(input *url.URL) *url.URL {
	if input == nil {
		return &url.URL{}
	}
	copy := *input
	return &copy
}

func (e *ReviewGatewayExecutor) runtimeForJob(job ReviewGatewayJob) (*common.RuntimeContext, error) {
	if e == nil || e.Runtime == nil || e.Runtime.Client == nil {
		return nil, fmt.Errorf("GitLink runtime is required")
	}
	installation, exists := e.Installations[job.InstallationID]
	if !exists {
		if e.RequireInstallationRuntime {
			return nil, fmt.Errorf("GitLink installation %q is not available", job.InstallationID)
		}
		return e.Runtime, nil
	}
	if !installation.Enabled {
		return nil, fmt.Errorf("GitLink installation %q is disabled", installation.InstallationID)
	}
	credential := ""
	if job.PublicRead && !reviewGatewayActionAllowsPublicRepository(job.Action) {
		return nil, fmt.Errorf("public repository discovery is not allowed for action %q", job.Action)
	}
	if !job.PublicRead && installation.CredentialRef != "" {
		var err error
		credential, err = resolveReviewGatewaySecretReference(installation.CredentialRef)
		if err != nil {
			return nil, fmt.Errorf("resolve GitLink installation credential: %w", err)
		}
	}
	runtimeCopy := *e.Runtime
	clientCopy := *e.Runtime.Client
	baseURL := strings.TrimRight(strings.TrimSpace(installation.GitLinkHost), "/")
	if baseURL != "" {
		parsed, err := url.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("parse GitLink installation host: %w", err)
		}
		if parsed.Path == "" || parsed.Path == "/" {
			baseURL += "/api"
		}
		clientCopy.BaseURL = baseURL
	}
	var baseTransport http.RoundTripper = http.DefaultTransport
	timeout := time.Duration(0)
	if e.Runtime.Client.HTTP != nil {
		timeout = e.Runtime.Client.HTTP.Timeout
		if !job.PublicRead {
			switch transport := e.Runtime.Client.HTTP.Transport.(type) {
			case *internalAuth.Transport:
				if transport.Base != nil {
					baseTransport = transport.Base
				}
			case nil:
			default:
				baseTransport = transport
			}
		}
	}
	clientCopy.HTTP = &http.Client{
		Transport: &reviewGatewayStaticAuthTransport{base: baseTransport, credential: credential},
		Timeout:   timeout,
	}
	runtimeCopy.Client = &clientCopy
	return &runtimeCopy, nil
}

func reviewGatewayInstallationMap(bindings ReviewGatewayBindings) map[string]GitLinkInstallation {
	result := make(map[string]GitLinkInstallation, len(bindings.Installations))
	for _, installation := range bindings.Installations {
		result[installation.InstallationID] = installation
	}
	return result
}

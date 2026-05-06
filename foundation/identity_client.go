package foundation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type IdentityProvider interface {
	GetToken(scope string) (string, error)
}

type tokenEntry struct {
	accessToken  string
	refreshToken string
	expiresAt    time.Time
}

type IdentityClient struct {
	baseURL      string
	companyID    string
	clientID     string
	clientSecret string
	cache        sync.Map
	httpClient   *http.Client
}

type tokenRequest struct {
	GrantType    string `json:"grant_type"`
	ClientID     string `json:"client_id,omitempty"`
	ClientSecret string `json:"client_secret,omitempty"`
	Scope        string `json:"scope,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	CompanyID    string `json:"company_id"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

func NewIdentityClient(baseURL, companyID, clientID, clientSecret string) *IdentityClient {
	return &IdentityClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		companyID:    companyID,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient:   &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *IdentityClient) GetToken(scope string) (string, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return "", fmt.Errorf("scope is required")
	}

	key := c.cacheKey(scope)
	if value, ok := c.cache.Load(key); ok {
		entry := value.(tokenEntry)
		if !entry.isExpired() {
			return entry.accessToken, nil
		}
		if refreshed, err := c.refreshToken(scope, entry.refreshToken); err == nil {
			return refreshed.accessToken, nil
		}
	}

	entry, err := c.fetchClientCredentials(scope)
	if err != nil {
		return "", err
	}
	return entry.accessToken, nil
}

func (c *IdentityClient) cacheKey(scope string) string {
	return fmt.Sprintf("%s|%s|%s", c.companyID, c.clientID, scope)
}

func (e tokenEntry) isExpired() bool {
	return time.Now().After(e.expiresAt)
}

func (c *IdentityClient) refreshToken(scope, refreshToken string) (tokenEntry, error) {
	if strings.TrimSpace(refreshToken) == "" {
		return tokenEntry{}, fmt.Errorf("refresh token missing")
	}

	entry, err := c.requestToken(tokenRequest{
		GrantType:    "refresh_token",
		RefreshToken: refreshToken,
		CompanyID:    c.companyID,
	})
	if err != nil {
		return tokenEntry{}, err
	}
	c.cache.Store(c.cacheKey(scope), entry)
	return entry, nil
}

func (c *IdentityClient) fetchClientCredentials(scope string) (tokenEntry, error) {
	entry, err := c.requestToken(tokenRequest{
		GrantType:    "client_credentials",
		ClientID:     c.clientID,
		ClientSecret: c.clientSecret,
		Scope:        scope,
		CompanyID:    c.companyID,
	})
	if err != nil {
		return tokenEntry{}, err
	}
	c.cache.Store(c.cacheKey(scope), entry)
	return entry, nil
}

func (c *IdentityClient) requestToken(req tokenRequest) (tokenEntry, error) {
	form := url.Values{}
	form.Set("grant_type", req.GrantType)
	form.Set("company_id", req.CompanyID)
	if req.ClientID != "" {
		form.Set("client_id", req.ClientID)
	}
	if req.ClientSecret != "" {
		form.Set("client_secret", req.ClientSecret)
	}
	if req.Scope != "" {
		form.Set("scope", req.Scope)
	}
	if req.RefreshToken != "" {
		form.Set("refresh_token", req.RefreshToken)
	}

	body := form.Encode()

	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodPost, c.baseURL+"/api/v1/oauth/token", strings.NewReader(body))
	if err != nil {
		return tokenEntry{}, err
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return tokenEntry{}, err
	}
	defer resp.Body.Close()

	responseBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return tokenEntry{}, fmt.Errorf("identity service error: status %d: %s", resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	var decoded tokenResponse
	if err := json.Unmarshal(responseBody, &decoded); err != nil {
		return tokenEntry{}, err
	}

	expiresIn := decoded.ExpiresIn - 30
	if expiresIn < 0 {
		expiresIn = 0
	}

	return tokenEntry{
		accessToken:  decoded.AccessToken,
		refreshToken: decoded.RefreshToken,
		expiresAt:    time.Now().Add(time.Duration(expiresIn) * time.Second),
	}, nil
}

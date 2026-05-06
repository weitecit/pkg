package foundation

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func decodeTokenRequest(t *testing.T, r *http.Request) tokenRequest {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var req tokenRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return req
}

func TestIdentityClientGetTokenUsesClientCredentialsAndCachesToken(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/api/v1/oauth/token" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		req := decodeTokenRequest(t, r)
		if req.GrantType != "client_credentials" {
			t.Fatalf("unexpected grant type: %s", req.GrantType)
		}
		if req.ClientID != "client-1" || req.ClientSecret != "secret-1" || req.CompanyID != "company-1" || req.Scope != "read" {
			t.Fatalf("unexpected request body: %+v", req)
		}
		_, _ = w.Write([]byte(`{"access_token":"access-1","refresh_token":"refresh-1","expires_in":40,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	client := NewIdentityClient(server.URL, "company-1", "client-1", "secret-1")
	got, err := client.GetToken("read")
	if err != nil {
		t.Fatalf("GetToken(): %v", err)
	}
	if got != "access-1" {
		t.Fatalf("token = %q, want %q", got, "access-1")
	}

	entryValue, ok := client.cache.Load("company-1|client-1|read")
	if !ok {
		t.Fatal("cache entry not stored")
	}
	entry := entryValue.(tokenEntry)
	remaining := time.Until(entry.expiresAt)
	if remaining < 5*time.Second || remaining > 15*time.Second {
		t.Fatalf("cache expiry remaining = %s, want around 10s", remaining)
	}

	got, err = client.GetToken("read")
	if err != nil {
		t.Fatalf("GetToken cached(): %v", err)
	}
	if got != "access-1" {
		t.Fatalf("cached token = %q, want %q", got, "access-1")
	}
	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount)
	}
}

func TestIdentityClientGetTokenRefreshesExpiredToken(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		req := decodeTokenRequest(t, r)
		if requestCount != 1 {
			t.Fatalf("unexpected request count: %d", requestCount)
		}
		if req.GrantType != "refresh_token" {
			t.Fatalf("unexpected grant type: %s", req.GrantType)
		}
		if req.RefreshToken != "refresh-old" {
			t.Fatalf("unexpected refresh token: %s", req.RefreshToken)
		}
		_, _ = w.Write([]byte(`{"access_token":"access-new","refresh_token":"refresh-new","expires_in":120,"token_type":"Bearer"}`))
	}))
	defer server.Close()

	client := NewIdentityClient(server.URL, "company-1", "client-1", "secret-1")
	client.cache.Store("company-1|client-1|read", tokenEntry{
		accessToken:  "access-old",
		refreshToken: "refresh-old",
		expiresAt:    time.Now().Add(-time.Minute),
	})

	got, err := client.GetToken("read")
	if err != nil {
		t.Fatalf("GetToken(): %v", err)
	}
	if got != "access-new" {
		t.Fatalf("token = %q, want %q", got, "access-new")
	}

	entryValue, ok := client.cache.Load("company-1|client-1|read")
	if !ok {
		t.Fatal("cache entry not stored")
	}
	entry := entryValue.(tokenEntry)
	if entry.accessToken != "access-new" || entry.refreshToken != "refresh-new" {
		t.Fatalf("cache entry = %+v, want refreshed tokens", entry)
	}
	if time.Until(entry.expiresAt) < 60*time.Second {
		t.Fatalf("refreshed expiry too short: %s", time.Until(entry.expiresAt))
	}
	if requestCount != 1 {
		t.Fatalf("requestCount = %d, want 1", requestCount)
	}
}

func TestIdentityClientGetTokenFallsBackToClientCredentialsWhenRefreshFails(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		req := decodeTokenRequest(t, r)
		switch requestCount {
		case 1:
			if req.GrantType != "refresh_token" {
				t.Fatalf("unexpected first grant type: %s", req.GrantType)
			}
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
		case 2:
			if req.GrantType != "client_credentials" {
				t.Fatalf("unexpected second grant type: %s", req.GrantType)
			}
			_, _ = w.Write([]byte(`{"access_token":"access-fallback","refresh_token":"refresh-fallback","expires_in":90,"token_type":"Bearer"}`))
		default:
			t.Fatalf("unexpected request count: %d", requestCount)
		}
	}))
	defer server.Close()

	client := NewIdentityClient(server.URL, "company-1", "client-1", "secret-1")
	client.cache.Store("company-1|client-1|read", tokenEntry{
		accessToken:  "access-old",
		refreshToken: "refresh-old",
		expiresAt:    time.Now().Add(-time.Minute),
	})

	got, err := client.GetToken("read")
	if err != nil {
		t.Fatalf("GetToken(): %v", err)
	}
	if got != "access-fallback" {
		t.Fatalf("token = %q, want %q", got, "access-fallback")
	}
	entryValue, ok := client.cache.Load("company-1|client-1|read")
	if !ok {
		t.Fatal("cache entry not stored")
	}
	entry := entryValue.(tokenEntry)
	if entry.accessToken != "access-fallback" || entry.refreshToken != "refresh-fallback" {
		t.Fatalf("cache entry = %+v, want fallback tokens", entry)
	}
	if requestCount != 2 {
		t.Fatalf("requestCount = %d, want 2", requestCount)
	}
}

func TestIdentityClientGetTokenRejectsEmptyInputs(t *testing.T) {
	client := NewIdentityClient("", "", "", "")
	if _, err := client.GetToken("   "); err == nil || !strings.Contains(err.Error(), "scope is required") {
		t.Fatalf("expected scope validation error, got %v", err)
	}
}

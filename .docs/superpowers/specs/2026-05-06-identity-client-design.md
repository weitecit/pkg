# Identity Client Design

**Date:** 2026-05-06
**Issue:** [#1 — feat(identity): JWT token wrapper with caching for Identity Service OAuth](../../issues/1)
**Status:** Approved

---

## Goal

Provide a reusable, cacheable OAuth2 client for the Weitec Identity Service. Consumers obtain a valid access token by calling a single method; the client handles token acquisition, caching, refresh, and fallback transparently.

---

## Architecture

### Location

`foundation/identity_client.go` — consistent with existing auth helpers in `foundation/gcp_auth.go`.

### Dependencies

Standard library only: `context`, `encoding/json`, `fmt`, `net/http`, `sync`, `time`. No external modules.

---

## Public API

```go
// IdentityProvider allows consumers to mock the client in tests.
type IdentityProvider interface {
    GetToken(ctx context.Context, scope string) (string, error)
}

// NewIdentityClient constructs a ready-to-use client.
func NewIdentityClient(baseURL, companyID, clientID, clientSecret string) *IdentityClient
```

`IdentityClient` implements `IdentityProvider`.

---

## Internal Types

```go
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
    cache        sync.Map // key: "companyID|clientID|scope"
}
```

```go
// HTTP request body
type tokenRequest struct {
    GrantType    string `json:"grant_type"`
    ClientID     string `json:"client_id,omitempty"`
    ClientSecret string `json:"client_secret,omitempty"`
    Scope        string `json:"scope,omitempty"`
    RefreshToken string `json:"refresh_token,omitempty"`
    CompanyID    string `json:"company_id"`
}

// HTTP response body
type tokenResponse struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    TokenType    string `json:"token_type"`
}
```

---

## GetToken Flow

```
1. Build cache key: fmt.Sprintf("%s|%s|%s", companyID, clientID, scope)

2. Look up key in sync.Map
   a. Entry not found           → go to step 4 (client_credentials)
   b. Entry found, token valid  → return accessToken directly
   c. Entry found, token expired → go to step 3 (refresh)

3. Attempt refresh_token grant
   POST /api/v1/oauth/token
   { "grant_type": "refresh_token", "refresh_token": <cached>, "company_id": <companyID> }
   - Success → overwrite cache entry → return new accessToken
   - Failure → continue to step 4

4. client_credentials grant
   POST /api/v1/oauth/token
   { "grant_type": "client_credentials", "client_id": ..., "client_secret": ...,
     "scope": ..., "company_id": ... }
   - Success → write cache entry → return accessToken
   - Failure → return error
```

### Token Expiry Calculation

```go
expiresAt = time.Now().Add(time.Duration(resp.ExpiresIn-30) * time.Second)
```

A 30-second buffer ensures the cached token is still valid when it reaches the downstream service.

### Validity Check

```go
func (e *tokenEntry) isExpired() bool {
    return time.Now().After(e.expiresAt)
}
```

---

## Cache Eviction Policy

Cache entries are **never evicted automatically**. An entry is only replaced when:

- A successful `refresh_token` grant returns new tokens.
- A successful `client_credentials` grant returns new tokens.

This means the `refreshToken` stays available even after the `accessToken` has expired, enabling the refresh flow on the next call.

---

## Concurrency

`sync.Map` handles concurrent reads and writes safely across goroutines. No additional mutex is used. Two goroutines may occasionally both execute a refresh or credentials call simultaneously; the last writer wins. Both obtained tokens are valid, so the redundant call is an acceptable trade-off for simplicity over single-flight complexity.

---

## HTTP Client

Uses `http.DefaultClient` with a 30-second timeout. No injected `http.Client` for now (YAGNI). If timeout configurability is needed in the future, it can be added via a constructor option without breaking the interface.

---

## Error Handling

- Non-2xx HTTP responses return a descriptive error including status code and response body.
- JSON decode failures return the decode error.
- Errors from the refresh step are silently swallowed internally; the client falls back to `client_credentials` and only surfaces an error if that also fails.

---

## Testing

`IdentityProvider` interface allows full mocking in unit tests for consumers. Integration tests for `IdentityClient` itself will use `httptest.NewServer` to simulate the Identity Service endpoint.

---

## Files Affected

| File | Change |
|------|--------|
| `foundation/identity_client.go` | New file — interface, struct, constructor, GetToken |
| `foundation/identity_client_test.go` | New file — unit + integration tests |

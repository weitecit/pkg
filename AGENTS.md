# Repository Instructions

## Quick facts
- This module is `github.com/weitecit/pkg` (`go 1.25.1`). It is a shared library, not an app: there is no `package main` entrypoint in the repo.
- The real package boundaries are `foundation/` (core models, repository abstractions, Mongo implementation), `services/` (request-building and auth/email helpers), `controllers/` (Gin-facing request/response helpers), `log/`, `utils/`, and `testutils/`.
- `controllers` depends on `services`, and `services` depends on `foundation`. Keep changes aligned with that direction instead of adding cross-package shortcuts.

## Verification commands
- Focused checks that are already green here: `go test ./foundation -run TestIdentityClient` and `go test ./services`.
- Whole-module `go vet ./...` is not a clean gate today. It currently reports `foundation/mongo_repository.go:68:9: return copies lock value`.
- `gofmt -l .` currently lists many files across the repo. Do not mass-format unrelated files as part of a small change.
- There is no checked-in CI workflow, `Makefile`, `Taskfile`, pre-commit config, or repo-local OpenCode config. Use direct Go commands instead of assuming wrapper scripts exist.

## Testing quirks
- `testutils.NewTestMongoServer` does **not** default to in-memory Mongo. With `MONGO_REPO` unset it tries `mongodb://localhost:27017/`.
- `go test ./testutils -run TestNewTestMongoServer` will panic if local Mongo is unavailable, because `NewTestMongoServer` calls `server.Stop()` on a nil `memongo` server after a failed ping.
- `MONGO_REPO=memory` is also not a safe fallback on Windows: `testutils/testutils.go` hardcodes a macOS MongoDB download URL for `memongo`.
- `services/services_test.go` sets baseline env for that package in `TestMain`: `ENVIRONMENT=test`, `SYSTEM_USER`, `SYSTEM_TOKEN`, and `DEFAULT_DATABASE`.
- Many auth/email tests in `services/system_service_test.go` depend on env vars such as `SECRET_KEY`, `OAUTH_CLIENT_ID`, `OAUTH_CLIENT_SECRET`, `OAUTH_TENANT_ID`, `MICROSOFT_CLIENT`, and `LANDING_URI`, but the tests usually set them themselves.

## Codebase gotchas
- There are still intentional gaps in `services/services.go`: `NewBaseRequestFromServiceRequestWithIDs` and `NewBaseRequestFromServiceRequest` only log/print `"Not implemented"` and return `nil` values. Do not assume those helpers are safe to wire into new paths.
- `foundation.NewRepositoryFromModel` is the normal repository construction path; it derives collection/global flags from the model and then delegates to MongoDB-only `NewRepository`.
- `log/sendDiscordMessage` is environment-sensitive: it short-circuits in `ENVIRONMENT=local` or `test`, so tests should keep one of those values when they must avoid real webhook sends.

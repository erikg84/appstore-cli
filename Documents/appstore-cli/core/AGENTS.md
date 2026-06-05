# AGENTS.md — core

The `core` module is the shared foundation imported by both `cli` and `server`.
Module path: `github.com/dallaslabs/appctl/core`

## Package Layout

| Package | Responsibility |
|---|---|
| `config` | App registry and credential loading (env vars with hardcoded defaults) |
| `store` | Domain model structs — **single source of truth for all data shapes** |
| `asc` | App Store Connect REST API client (ES256 JWT auth) |
| `play` | Google Play API clients (OAuth2 service account) |

## Rules

- **Never import `cli` or `server` packages here.** Core is a leaf dependency.
- **Always update `store/models.go` first** when adding new data. Downstream modules depend on these types.
- `asc/client.go` contains shared helpers (`dataItems`, `str`, `intVal`, `truncate`) — use them instead of duplicating JSON parsing.
- `play/client.go:NewClient()` creates a single shared `oauth2.TokenSource` used by androidpublisher, playdeveloperreporting, and storage/v1 — do not create separate credentials for each service.

## Google Play API Gotchas

- `Inappproducts.List` → **deprecated, returns 403**. Use `Monetization.Onetimeproducts.List`.
- `androidpublisher` has no app listing endpoint. Use `play/client.go:ListApps()` via Reporting API.
- GCS reports are UTF-16 LE/BE inside ZIP files. `play/reports.go:decodeUTF16()` handles this.
- Required OAuth2 scopes: `androidpublisher` + `playdeveloperreporting` + `devstorage.read_only`.

## App Store Connect API Gotchas

- JWT tokens expire in 20 minutes — regenerate per call (already handled in `asc/client.go`).
- List responses wrap items in `data[]` — use `dataItems()` helper.
- Sales reports are gzip TSV — use `asc/reports.go`.
- Analytics report requests:
  - Do **not** rely on `GET /v1/analyticsReportRequests` (global collection is forbidden for this resource in current ASC behavior).
  - Use app-scoped listing `GET /v1/apps/{id}/analyticsReportRequests` and create with `POST /v1/analyticsReportRequests`.
  - iOS installs/downloads require one-time `ONGOING` request setup per app before instances appear.

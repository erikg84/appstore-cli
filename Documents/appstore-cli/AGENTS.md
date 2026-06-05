# AGENTS.md — appstore-cli

Guidance for AI agents (GitHub Copilot, etc.) working in this repository.

---

## Module Map

| Module | Path | Role |
|---|---|---|
| `core` | `github.com/dallaslabs/appctl/core` | Shared API clients, models, config — imported by cli and server |
| `cli` | `github.com/dallaslabs/appctl/cli` | Cobra CLI binary (`appctl`) |
| `server` | `github.com/dallaslabs/appctl/server` | Chi REST + gqlgen GraphQL server (`appctl-server`) |

All three are tied together by `go.work` at the repo root. **Do not import `cli` or `server` from `core`.**

---

## Key Files

| File | Purpose |
|---|---|
| `core/config/config.go` | App registry (aliases → ASC IDs + Play packages) and credential defaults |
| `core/store/models.go` | **Canonical domain models** — add fields here first, then update clients/handlers |
| `core/asc/client.go` | App Store Connect JWT client + core methods |
| `core/asc/iap.go` | ASC in-app purchases |
| `core/asc/subscriptions.go` | ASC subscription groups |
| `core/asc/testflight.go` | TestFlight beta groups + testers |
| `core/asc/reports.go` | ASC sales reports (gzip TSV) |
| `core/asc/users.go` | ASC team users |
| `core/play/client.go` | Play OAuth2 SA client + `ListApps()` via Reporting API |
| `core/play/iap.go` | Play one-time products (`Monetization.Onetimeproducts`) |
| `core/play/subscriptions.go` | Play subscriptions (`Monetization.Subscriptions`) |
| `core/play/reports.go` | GCS pubsite bucket access (earnings, sales, stats CSVs) |
| `core/play/users.go` | Play Console users |
| `server/rest/router.go` | All REST route definitions |
| `server/rest/handlers.go` | REST handler implementations |
| `server/graphql/schema.graphql` | GraphQL schema — source of truth for queryable types |
| `server/graphql/schema.resolvers.go` | GraphQL resolver implementations |
| `cli/cmd/root.go` | Global flags: `--output`, `--server` |

---

## Credentials & Environment

Never hardcode secrets. Credentials load from env vars with fallback defaults in `config.go`:

| Env var | Default | Description |
|---|---|---|
| `ASC_KEY_FILE` | `~/Downloads/AuthKey_746FSCD2PK.p8` | ASC ES256 primary key (Admin + Finance) |
| `ASC_KEY_ID` | `746FSCD2PK` | ASC primary key ID |
| `ASC_ISSUER_ID` | `e68d90fb-c9b8-41dd-ad08-947afe7459ae` | ASC issuer ID |
| `ASC_ANALYTICS_KEY_FILE` | `~/Downloads/AuthKey_79PWD5WB3S.p8` | ASC analytics-only key for installs/downloads path |
| `ASC_ANALYTICS_KEY_ID` | `79PWD5WB3S` | ASC analytics key ID |
| `ASC_ANALYTICS_ISSUER_ID` | `ASC_ISSUER_ID` | Optional override for analytics issuer |
| `ASC_VENDOR_ID` | `92547182` | ASC vendor number for sales reports |
| `PLAY_KEY_FILE` | `~/Downloads/play-service-account.json` | GCP service account JSON |
| `PLAY_DEVELOPER_ACCOUNT` | *(not set)* | Play developer account ID (for users API) |

> An older key `A5Q22QH9WG` (`~/Downloads/AuthKey_A5Q22QH9WG.p8`) exists but has **no Finance access**. The active key is `746FSCD2PK`.

---

## Adding a New App

Edit `core/config/config.go`, add an entry to the `Apps` map:

```go
"my-app": {
    Name:        "My App",
    ASCAppID:    "1234567890",
    PlayPackage: "com.example.myapp",
},
```

No other files need changing — the alias propagates automatically to CLI and server.

---

## Adding a New API Domain

1. Add model(s) to `core/store/models.go`
2. Add client method(s) to the relevant `core/asc/*.go` or `core/play/*.go`
3. Add CLI command in `cli/cmd/<domain>.go` and register it in `cli/cmd/root.go`
4. Add REST handler in `server/rest/handlers.go`, route in `server/rest/router.go`
5. Add GraphQL type + field to `server/graphql/schema.graphql`, regenerate, implement resolver

---

## Google Play API Notes

- **`androidpublisher` v3 has no list-all-apps endpoint.** Use `core/play/client.go:ListApps()` which calls `playdeveloperreporting/v1beta1/apps:search`.
- **IAP**: Use `Monetization.Onetimeproducts` — the old `Inappproducts` endpoint returns HTTP 403.
- **Subscriptions**: Use `Monetization.Subscriptions`.
- **GCS reports**: Bucket is `pubsite_prod_5992512819868906729`. Files are UTF-16 LE/BE encoded or plain UTF-8 inside ZIPs. `core/play/reports.go` handles decode automatically.
- **Scopes needed**: `androidpublisher`, `playdeveloperreporting`, `devstorage.read_only` — all requested in `NewClient()`.

---

## App Store Connect API Notes

- Auth: ES256 JWT, 20-minute expiry. Regenerated per request in `core/asc/client.go`.
- Pagination: Use `?limit=200` where possible. `dataItems()` helper in `client.go` handles the `data[]` wrapper.
- Sales reports: require `Accept: application/a-gzip` (NOT `application/json`). Returns gzip-compressed TSV. `core/asc/reports.go` decompresses and parses. Vendor number `92547182` is the default in config — no need to pass `--vendor` from CLI.
- Finance access requires the `746FSCD2PK` key — the older `A5Q22QH9WG` key will 403 on `/v1/salesReports` and `/v1/financeReports`.
- Analytics installs/downloads flow uses app-scoped requests: `GET /v1/apps/{id}/analyticsReportRequests` and `POST /v1/analyticsReportRequests`.
- The global collection endpoint `GET /v1/analyticsReportRequests` is forbidden (`GET_COLLECTION` not allowed).
- iOS analytics requires one-time `ONGOING` request setup per app; until Apple generates instances, installs can return `no analytics instances found`.

---

## Build & Test

```bash
# Build everything
cd ~/Documents/appstore-cli
go build -C cli    -o appctl        .
go build -C server -o appctl-server .

# Install globally
sudo cp appctl appctl-server /usr/local/bin/

# Run server locally
ASC_KEY_FILE=~/Downloads/AuthKey_746FSCD2PK.p8 \
ASC_ANALYTICS_KEY_FILE=~/Downloads/AuthKey_79PWD5WB3S.p8 \
PLAY_KEY_FILE=~/Downloads/play-service-account.json \
./appctl-server --port 8080

# Smoke test
curl http://localhost:8080/health
curl http://localhost:8080/api/v1/apps
appctl apps
appctl reviews venus --store both
appctl reports play files --category earnings
appctl reports sales --date 2025-12
```

---

## GCS Bucket Layout

```
gs://pubsite_prod_5992512819868906729/
├── earnings/          earnings_YYYYMM_*.zip  (per-transaction, UTF-16 CSV in ZIP)
├── sales/             salesreport_YYYYMM.zip (order-level, UTF-16 CSV in ZIP)
├── reviews/           reviews_{package}_{YYYYMM}.csv
├── stats/
│   ├── crashes/       crashes_{package}_{YYYYMM}_{dimension}.csv
│   └── installs/      installs_{package}_{YYYYMM}_{dimension}.csv
└── acquisition/
    ├── buyers_7d/
    └── retained_installers/
```

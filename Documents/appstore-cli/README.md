# appctl — Unified App Store CLI & Server

A single Go tool to manage **Apple App Store Connect** and **Google Play Console** from one place.
Includes a full CLI (`appctl`) and a REST + GraphQL server (`appctl-server`).

---

## Contents

- [Architecture](#architecture)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [CLI Reference](#cli-reference)
- [Server Reference](#server-reference)
- [GraphQL API](#graphql-api)
- [Building from Source](#building-from-source)

---

## Architecture

```
appstore-cli/
├── core/           # Shared Go module — API clients, models, config
│   ├── asc/        # App Store Connect REST client (JWT/ES256)
│   ├── play/       # Google Play + Reporting API + GCS client (OAuth2 SA)
│   ├── config/     # App registry + credential loading
│   └── store/      # Shared domain models
├── cli/            # Cobra CLI binary (appctl)
├── server/         # Chi REST + gqlgen GraphQL server (appctl-server)
└── go.work         # Go workspace tying all three modules together
```

`cli` and `server` are independent modules that both import `core` — no code duplication.

---

## Prerequisites

| Credential | File | Env var |
|---|---|---|
| ASC primary key (.p8) | `~/Downloads/AuthKey_746FSCD2PK.p8` | `ASC_KEY_FILE` |
| ASC analytics key (.p8) | `~/Downloads/AuthKey_79PWD5WB3S.p8` | `ASC_ANALYTICS_KEY_FILE` |
| Play service account key | `~/Downloads/play-service-account.json` | `PLAY_KEY_FILE` |

The defaults are hardcoded in `core/config/config.go` for local use. Override via env vars.

**ASC credentials:**
- Primary key ID: `746FSCD2PK` (Admin + Finance access; default for most ASC endpoints)
- Analytics key ID: `79PWD5WB3S` (optional analytics-only key for installs/downloads path)
- Issuer ID: `e68d90fb-c9b8-41dd-ad08-947afe7459ae`
- Vendor Number: `92547182` (default, override with `ASC_VENDOR_ID`)

> An older key `A5Q22QH9WG` exists in `~/Downloads` but has no Finance access. Use `746FSCD2PK`.

**Play credentials:**
- Service account: `fastlane-deploy@venus-companion.iam.gserviceaccount.com`
- GCP project: `venus-companion`
- GCS pubsite bucket: `gs://pubsite_prod_5992512819868906729/`

---

## Installation

Both binaries are installed at `/usr/local/bin/`:

```bash
which appctl          # /usr/local/bin/appctl
which appctl-server   # /usr/local/bin/appctl-server
```

To reinstall after rebuilding:

```bash
cd ~/Documents/appstore-cli
go build -C cli    -o appctl        .
go build -C server -o appctl-server .
sudo cp appctl appctl-server /usr/local/bin/
```

---

## Configuration

Apps are registered in `core/config/config.go`:

| Alias | App Store Connect ID | Google Play Package |
|---|---|---|
| `venus` | 6754580293 | com.dallaslabs.venus |
| `polity-now` | 6758637143 | org.dallaslabs.estado |
| `iss-spotter` | 6757094244 | org.dallaslabs.issspotter |
| `quake` | 6758176095 | org.dallaslabs.quakealert |
| `cabinet-doors` | 6755406345 | org.dallaslabs.cabinetdoors.cabinetdoors |
| `snake` | 6472907910 | — |
| `kmp-demo` | 6759348156 | — |

---

## CLI Reference

All commands support `--output table|json` and `--server <url>` (proxy to running server).

### Global flags

```
--output string   Output format: table or json (default "table")
-o string         Shorthand for --output
--server string   Proxy all commands through an appctl-server base URL
--no-header       Suppress column headers in table output
```

### apps

```bash
appctl apps                   # List all configured apps
appctl apps --output json
```

### versions

```bash
appctl versions <alias>       # iOS/macOS versions for an app
appctl versions venus
appctl versions venus --output json
```

### builds

```bash
appctl builds <alias>         # TestFlight builds
appctl builds venus
```

### tracks

```bash
appctl tracks <alias>         # Google Play release tracks
appctl tracks venus
```

### reviews

```bash
appctl reviews <alias>                    # Reviews from both stores
appctl reviews venus --store ios          # iOS only
appctl reviews venus --store android      # Android only
appctl reviews venus --store both         # Explicit both (default)
```

### installs

```bash
appctl installs <alias>                   # Combined iOS + Android install stats
appctl installs venus --store android 202512
appctl installs venus --store android --breakdown country 202512
appctl installs venus --store ios --output json
```

For iOS install/download stats, ASC analytics requires one-time `ONGOING`
analytics report requests per app. If requests exist but no report instances
have been generated yet, iOS installs can return `no analytics instances found`.

### iap

```bash
appctl iap list <alias>                   # In-app purchases (both stores)
appctl iap list venus --store ios
appctl iap list venus --store android
```

### subscriptions

```bash
appctl subscriptions list <alias>         # Subscriptions (both stores)
appctl subscriptions list venus --store ios
appctl subscriptions list venus --store android
```

### testflight

```bash
appctl testflight groups <alias>          # Beta groups
appctl testflight testers <alias>         # Beta testers
appctl testflight groups venus
```

### reports

```bash
# App Store Connect sales report (vendor defaults to 92547182)
appctl reports sales --date 2025-12
appctl reports sales --date 2025-12 --frequency MONTHLY
appctl reports sales --date 2025-12 --output json
appctl reports sales --date 2025-12 --vendor 92547182   # explicit override

# Google Play GCS reports
appctl reports play files                             # List all available files
appctl reports play files --category earnings         # Filter by category
appctl reports play files --category sales
appctl reports play files --category reviews
appctl reports play files --category "stats/crashes"
appctl reports play files --category "stats/installs"

appctl reports play earnings          # Latest earnings (auto-detects newest file)
appctl reports play earnings 202604   # Specific YYYYMM
appctl reports play sales
appctl reports play sales 202512
appctl reports play installs venus 202512
appctl reports play installs venus 202512 --breakdown country
appctl reports play crashes iss-spotter
appctl reports play acquisition cabinet-doors --type retained_installers
```

Play report categories in GCS bucket:

| Category | Description |
|---|---|
| `earnings/` | Per-transaction earnings ZIPs |
| `sales/` | Order-level sales ZIPs |
| `reviews/` | App review CSVs |
| `stats/crashes/` | Crash stats by version/device/OS |
| `stats/installs/` | Install stats by country/carrier/device |
| `acquisition/` | Buyers and retained installer stats |

### users

```bash
appctl users list             # App Store Connect + Play Console users
appctl users list --output json
```

---

## Server Reference

Start the server:

```bash
appctl-server                           # Default port 8080
appctl-server --port 9090

# With explicit credential paths:
ASC_KEY_FILE=~/Downloads/AuthKey_746FSCD2PK.p8 \
ASC_ANALYTICS_KEY_FILE=~/Downloads/AuthKey_79PWD5WB3S.p8 \
PLAY_KEY_FILE=~/Downloads/play-service-account.json \
appctl-server --port 8080
```

The server exposes REST at `/api/v1/` and GraphQL at `/graphql`.

### REST Endpoints

#### Apps
| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/apps` | List all configured apps |
| GET | `/api/v1/apps/{alias}/versions` | App Store versions |
| GET | `/api/v1/apps/{alias}/builds` | TestFlight builds |
| GET | `/api/v1/apps/{alias}/tracks` | Play release tracks |
| GET | `/api/v1/apps/{alias}/reviews?store=ios\|android\|both` | Reviews |
| GET | `/api/v1/apps/{alias}/installs?store=&month=&breakdown=` | Install/download stats |
| GET | `/api/v1/apps/{alias}/iap?store=ios\|android\|both` | In-app purchases |
| GET | `/api/v1/apps/{alias}/subscriptions?store=ios\|android\|both` | Subscriptions |
| GET | `/api/v1/apps/{alias}/testflight/groups` | Beta groups |
| GET | `/api/v1/apps/{alias}/testflight/testers` | Beta testers |

#### Reports
| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/reports/sales?vendor=&date=&frequency=` | ASC sales report |
| GET | `/api/v1/reports/play/files?category=` | List Play GCS report files |
| GET | `/api/v1/reports/play/earnings?month=YYYYMM` | Download earnings from GCS |
| GET | `/api/v1/reports/play/sales?month=YYYYMM` | Download sales from GCS |
| GET | `/api/v1/reports/play/installs?package=&month=&breakdown=` | Play install stats |
| GET | `/api/v1/reports/play/crashes?package=&month=&breakdown=` | Play crash stats |
| GET | `/api/v1/reports/play/acquisition?package=&month=&type=` | Play acquisition stats |

#### Users & Health
| Method | Path | Description |
|---|---|---|
| GET | `/api/v1/users` | ASC + Play users |
| GET | `/health` | Server health check |

### Using the CLI against the server

```bash
appctl --server http://localhost:8080 apps
appctl --server http://localhost:8080 reviews venus --store both
appctl --server http://localhost:8080 reports play files --category earnings
```

---

## GraphQL API

Endpoint: `POST http://localhost:8080/graphql`

### Example queries

```graphql
# List all apps
{ apps { alias name ascId playPackage } }

# Versions and tracks for an app
{
  versions(alias: "venus") { platform versionString state }
  tracks(alias: "venus") { name status versionCodes }
}

# Reviews across both stores
{ reviews(alias: "venus", store: "both") { store rating title body date } }

# TestFlight groups
{ betaGroups(alias: "venus") { name isInternal testerCount } }

# In-app purchases
{ iap(alias: "venus", store: "ios") { productId name price state } }

# Subscriptions
{ subscriptions(alias: "venus", store: "both") { productId name state } }
```

---

## Building from Source

```bash
cd ~/Documents/appstore-cli

# Build both binaries
go build -C cli    -o appctl        .
go build -C server -o appctl-server .

# Install globally
sudo cp appctl appctl-server /usr/local/bin/

# Build and run server in one step
go run ./server --port 8080
```

Go workspace: requires Go 1.22+. All three modules (`core`, `cli`, `server`) are managed via `go.work`.

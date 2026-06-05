# AGENTS.md — cli

The `cli` module builds the `appctl` binary using Cobra.
Module path: `github.com/dallaslabs/appctl/cli`
Installed at: `/usr/local/bin/appctl`

## Command Structure

```
appctl
├── apps                          List configured apps
├── versions <alias>              App Store versions
├── builds <alias>                TestFlight builds
├── tracks <alias>                Google Play release tracks
├── reviews <alias>               Reviews (--store ios|android|both)
├── installs <alias> [YYYYMM]     Install/download stats (--store, --breakdown)
├── iap
│   └── list <alias>             In-app purchases (--store)
├── subscriptions
│   └── list <alias>             Subscriptions (--store)
├── testflight
│   ├── groups <alias>           Beta groups
│   └── testers <alias>          Beta testers
├── reports
│   ├── sales                    ASC sales (--date, --frequency, optional --vendor)
│   └── play
│       ├── files                List GCS report files (--category)
│       ├── earnings [YYYYMM]    Download earnings from GCS
│       ├── sales [YYYYMM]       Download sales from GCS
│       ├── installs <app> [YYYYMM]
│       ├── crashes <app> [YYYYMM]
│       └── acquisition <app> [YYYYMM]
└── users
    └── list                     ASC + Play users
```

## Global Flags

- `--output table|json` / `-o` — controls output format (default: table)
- `--server <url>` — proxy all calls through a running `appctl-server`
- `--no-header` — suppress column headers (useful for piping/scripting)

## Adding a Command

1. Create `cli/cmd/<domain>.go`
2. Define a `newXxxCmd()` function returning `*cobra.Command`
3. Register it in `cli/cmd/root.go` with `rootCmd.AddCommand(newXxxCmd())`
4. Use `ascClient()` / `playClient()` helpers from `helpers.go` — do not construct clients inline
5. Use `ascAnalyticsClient()` for iOS analytics install/download endpoints
6. Use `render(data, tableFunc)` for dual table/JSON output
7. Use `die(err)` to return errors with proper exit codes

## Conventions

- Table output: left-aligned columns with `fmt.Printf("%-Ns ...")` headers
- Use `trim(s, N)` helper to truncate long strings in table view
- For cross-store commands, accept `--store ios|android|both` flag (default: `both`)
- Use `--no-header` global flag to suppress headers; check `noHeader` bool in render funcs
- `appctl reports sales` defaults vendor number to `92547182` (from config) — `--vendor` is optional
- Commands that return lists should support `--limit N` where it makes sense (builds, reviews, testers)

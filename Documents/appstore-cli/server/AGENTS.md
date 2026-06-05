# AGENTS.md — server

The `server` module builds the `appctl-server` binary.
Module path: `github.com/dallaslabs/appctl/server`
Installed at: `/usr/local/bin/appctl-server`
Default port: `8080`

## Stack

- **HTTP router**: [go-chi/chi v5](https://github.com/go-chi/chi)
- **GraphQL**: [gqlgen](https://gqlgen.com/)
- REST and GraphQL share the same Chi router and port

## REST Route Map

```
GET  /health
GET  /api/v1/apps
GET  /api/v1/apps/{alias}/versions
GET  /api/v1/apps/{alias}/builds
GET  /api/v1/apps/{alias}/tracks
GET  /api/v1/apps/{alias}/reviews          ?store=ios|android|both
GET  /api/v1/apps/{alias}/installs         ?store=&month=&breakdown=
GET  /api/v1/apps/{alias}/iap              ?store=ios|android|both
GET  /api/v1/apps/{alias}/subscriptions    ?store=ios|android|both
GET  /api/v1/apps/{alias}/testflight/groups
GET  /api/v1/apps/{alias}/testflight/testers
GET  /api/v1/reports/sales                 ?vendor=&date=&frequency=
GET  /api/v1/reports/play/files            ?category=
GET  /api/v1/reports/play/earnings         ?month=YYYYMM
GET  /api/v1/reports/play/sales            ?month=YYYYMM
GET  /api/v1/reports/play/installs         ?package=&month=&breakdown=
GET  /api/v1/reports/play/crashes          ?package=&month=&breakdown=
GET  /api/v1/reports/play/acquisition      ?package=&month=&type=
GET  /api/v1/users
POST /graphql
```

## Adding a REST Endpoint

1. Add handler method to `server/rest/handlers.go` on the `Handler` struct
2. Register the route in `server/rest/router.go`
3. Use `writeJSON(w, status, data)` for success and `writeError(w, status, msg)` for errors
4. Use `appByAlias(chi.URLParam(r, "alias"))` to resolve app aliases
5. Use `ascClient()` / `playClient()` helpers at the top of `handlers.go`
6. Use `ascAnalyticsClient()` for iOS installs/download analytics paths

## Adding a GraphQL Field

1. Add the type/field to `server/graphql/schema.graphql`
2. Run `go run github.com/99designs/gqlgen generate` from `server/`
3. Implement the new method in `server/graphql/schema.resolvers.go`

## Middleware (applied to all /api/v1/ routes)

- `JSONContentType` — sets `Content-Type: application/json`
- `CORS` — allows all origins (local/trusted network only)
- `RequestLogger` — logs method, path, duration to stdout

## Security

No authentication by design — intended for local or trusted-network use only.
Do not expose `appctl-server` on a public interface without adding auth middleware.

# APIWeaver

> Composable building blocks for resilient Go APIs: consistent responders,
> self-documenting info endpoints, and pragmatic health probes.

## Table of Contents

- [APIWeaver](#apiweaver)
  - [Table of Contents](#table-of-contents)
  - [Highlights](#highlights)
  - [Packages at a Glance](#packages-at-a-glance)
  - [Install](#install)
  - [Quick Start](#quick-start)
  - [Examples](#examples)
  - [Router](#router)
  - [Health, Docs \& Probes](#health-docs--probes)
  - [Tooling \& Generation](#tooling--generation)
  - [Development](#development)

## Highlights

- Functional options everywhere; wire only the collaborators you care about.
- Shared `Responder` keeps JSON envelopes, trace IDs, and structured logs in
  sync across handlers.
- `InfoHandler` exposes `/status`, `/version`, and `/docs` (with an embedded
  OpenAPI viewer) in just a few lines.
- Probe adapters convert database/client checks into HTTP-friendly readiness
  handlers.
- Lightweight helpers (`jsonutil`) wrap [sonic](https://github.com/bytedance/sonic)
  for high-throughput encoding without sprinkling boilerplate through your code.

## Packages at a Glance

| Package | What it solves |
| --- | --- |
| `responder` | JSON rendering, error envelopes, request decoding, metadata + ULIDs. |
| `info` | Status/version endpoints, OpenAPI + AsyncAPI JSON + HTML viewers, build metadata. |
| `probe` | Ready-made checks for databases or custom closures wired to HTTP. |
| `jsonutil` | Tiny helpers around sonic for fast (un)marshalling. |
| `router` | ServeMux with OpenAPI validation, CORS, timeout, and logging defaults via functional options. |

Each package can be imported independently, keeping binaries trim and focused.

## Install

```bash
go get github.com/drblury/apiweaver/responder
go get github.com/drblury/apiweaver/info
go get github.com/drblury/apiweaver/probe
go get github.com/drblury/apiweaver/jsonutil
go get github.com/drblury/apiweaver/router
```

> Requires Go 1.21+ (module declares 1.25) so you can rely on the latest stdlib
> improvements and generics.

## Quick Start

```go
import (
    "net/http"

    "github.com/drblury/apiweaver/info"
    "github.com/drblury/apiweaver/probe"
    "github.com/drblury/apiweaver/responder"
)

resp := responder.NewResponder(
    responder.WithLogger(logger),
    responder.WithStatusMetadata(http.StatusBadRequest, responder.Metadata{
        Level: "warn",
        Label: "validation",
    }),
)

infoHandler := info.NewInfoHandler(
    info.WithInfoResponder(resp),
    info.WithBaseURL("https://api.example.com"),
    info.WithUIType(info.UIScalar), // Choose your preferred OpenAPI UI
    info.WithInfoProvider(func() any {
        return map[string]string{
            "version": version,
            "commit":  commit,
        }
    }),
    info.WithSwaggerProvider(func() ([]byte, error) {
        return embeddedSpec, nil
    }),
    info.WithReadinessChecks(
        probe.NewMongoPingProbe(mongoClient, nil),
    ),
)

mux := http.NewServeMux()
mux.HandleFunc("/status", infoHandler.GetStatus)
mux.HandleFunc("/version", infoHandler.GetVersion)
mux.HandleFunc("/docs", infoHandler.GetOpenAPIHTML)
mux.HandleFunc("/openapi.json", infoHandler.GetOpenAPIJSON)
```

Share the same `Responder` anywhere you need consistent tracing metadata or
error semantics (routers, middleware, background workers, etc.).

## Examples

Every package now includes runnable `Example*` functions demonstrating typical
and optional advanced integrations. Execute them directly via `go test`:

```bash
go test ./info     -run Example
go test ./jsonutil -run Example
go test ./responder -run Example
go test ./router   -run Example
go test ./probe    -run Example
```

Refer to the following table to jump into a concrete scenario:

| Package | Example Highlights |
| --- | --- |
| `info` | Wires `InfoHandler` with custom base URL, swagger provider, and probes, then exercises `/healthz` + `/version`. Also demonstrates AsyncAPI support for event-driven APIs. |
| `jsonutil` | Demonstrates struct marshal/unmarshal plus streaming `Encode`/`Decode`. |
| `responder` | End-to-end handler showing request decoding, validation, custom error classification, and structured problem payloads. |
| `router` | Builds a mux with OpenAPI validation, logger, tuned CORS, timeout, and prepend/append middlewares. |
| `probe` | Covers generic ping probes plus configurable HTTP checks (status windows, request mutators, response validators) via `NewHTTPProbe`. |

The examples double as documentation for `go doc`, so you can read and run them
without scaffolding a separate binary.

## Router

`router.New` wraps your generated handlers with a configurable middleware
stack—OpenAPI validation, CORS, per-request timeouts, and structured logging are
enabled by default and can be reordered or replaced via functional options.

```go
swagger, _ := openapi3.NewLoader().LoadFromFile("./internal/server/_gen/openapi.json")

mux := router.New(
  generatedHandler,
  router.WithSwagger(swagger),
  router.WithLogger(logger),
  router.WithConfig(router.Config{
    Timeout: 5 * time.Second,
    CORS: router.CORSConfig{
      Origins: []string{"https://app.example.com"},
      Methods: []string{"GET", "POST"},
      Headers: []string{"Content-Type", "Authorization"},
      AllowCredentials: true,
    },
    QuietdownRoutes: []string{"/status"},
    HideHeaders:     []string{"Authorization"},
  }),
  router.WithMiddlewares(metricsMiddleware),
)

http.ListenAndServe(":8080", mux)
```

Additional helpers (`router.WithTrailingMiddlewares`,
`router.WithMiddlewareChain`, `router.Without*`) make it easy to blend your own
middleware with the built-in defaults.

## Health, Docs & Probes

- **HTML docs**: Multiple OpenAPI documentation UIs are supported out of the box:
  - **Stoplight Elements** (default): `info/assets/stoplight.html`
  - **Scalar**: Modern, interactive API documentation
  - **SwaggerUI**: The classic OpenAPI documentation tool
  - **Redoc**: Clean, responsive OpenAPI documentation

  Use `info.WithUIType()` to select your preferred UI (e.g., `info.WithUIType(info.UIScalar)`).
- **AsyncAPI docs**: For event-driven APIs, AsyncAPI documentation is also supported:
  - Use `info.WithAsyncAPIProvider()` to supply your AsyncAPI spec
  - Use `info.WithAsyncAPITemplate()` for custom HTML templates
  - Use `info.WithAsyncAPITemplateData()` for custom template data
  - Default template uses [AsyncAPI React Component](https://github.com/asyncapi/asyncapi-react)
  - Use `info.AsyncAPISpecURL(baseURL)` helper to construct spec URLs
- **JSON docs**: Provide a `SwaggerProvider` (or `OpenAPIProvider`) to serve the
  raw spec alongside the viewer.
- **Readiness/Liveness**: Compose the built-in probes (`probe` package) or pass
  your own `func(context.Context) error` implementations. Failures are surfaced
  via the responder with correlation IDs intact.
- **Reverse proxies**: set `info.WithBaseURL` so generated links point to the
  external host.

## Tooling & Generation

- `task gen-api` (from `taskfile.yml`) runs the Dockerised `oapi-codegen`
  pipeline described in `generate.go` to keep server stubs current.
- Generated assets and the documentation viewer live under `info/assets/` and
  are embedded so you can ship a single binary.

## Development

```bash
# Run unit tests for every package
go test ./...

# Regenerate OpenAPI-based handlers and assets
task gen-api
```

- Keep `docs.go` (this repo) updated as packages evolve so `go doc` stays
  helpful.
- Pair `info` with your router package of choice for validation, logging, or
  other cross-cutting plugins.
- Contributions welcome—open an issue or PR before making large changes.

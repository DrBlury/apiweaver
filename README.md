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
| `info` | Status/version endpoints, OpenAPI JSON + HTML viewer, build metadata. |
| `probe` | Ready-made checks for databases or custom closures wired to HTTP. |
| `jsonutil` | Tiny helpers around sonic for fast (un)marshalling. |

Each package can be imported independently, keeping binaries trim and focused.

## Install

```bash
go get github.com/drblury/apiweaver/responder
go get github.com/drblury/apiweaver/info
go get github.com/drblury/apiweaver/probe
go get github.com/drblury/apiweaver/jsonutil
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

## Health, Docs & Probes

- **HTML docs**: An embedded Stoplight viewer (`info/assets/stoplight.html`)
  serves your OpenAPI spec without extra tooling.
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

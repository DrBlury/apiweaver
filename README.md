# API Package

## Overview

The `api` module now provides focused subpackages that can be pulled in
individually:

- `responder`: consistent JSON error envelopes and structured logging via
  `Responder`.
- `info`: lightweight health, version, and documentation endpoints powered by
  `InfoHandler`.
- `probe`: helper adapters for wiring database or custom checks into
  readiness/liveness probes.
- Tooling integration (`go:generate`) sits at the package root and keeps the
  generated API handlers in sync with the specification.

The package embraces the **functional options** pattern so callers can opt in to
optional collaborators without juggling long parameter lists.

## Key Components

### Responder

`Responder` (imported from `pkg/api/responder`) wraps common HTTP tasks such as
rendering JSON, decoding request bodies, and emitting structured error payloads.
Errors are enriched with a ULID, category, timestamp, and log metadata so they
remain traceable across systems. `WithStatusMetadata` lets you override the
logging level or error labels for individual HTTP status codes.

### InfoHandler

`InfoHandler` (imported from `pkg/api/info`) combines the generated handlers
with convenience endpoints that expose build metadata (`GET /version`), health
information (`GET /status`), and an HTML viewer for your OpenAPI document.
Collaborators are injected via `InfoOption` values so the handler can be
assembled with only the bits you need.

### Generation Workflow

The `generate.go` file wires `go generate` to a Dockerised instance of
`oapi-codegen`. Run `task gen-api` (as defined in `taskfile.yml`) whenever the
OpenAPI specification changes to refresh the generated server stubs. The
embedded HTML assets under `embedded/` are served by the info handler.

## Usage Example

```go
import (
  "html/template"
  "net/http"

  "github.com/drblury/apiweaver/api/info"
  "github.com/drblury/apiweaver/api/responder"
)

resp := responder.NewResponder(
  responder.WithLogger(logger),
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
  info.WithOpenAPITemplate(template.Must(template.ParseFiles("./docs/viewer.html"))),
  info.WithOpenAPITemplateData(func(_ *http.Request, baseURL string) any {
    return map[string]any{
      "BaseURL": baseURL,
      "SpecURL": baseURL + "/info/openapi.json",
    }
  }),
)

mux := http.NewServeMux()
mux.HandleFunc("/status", infoHandler.GetStatus)
mux.HandleFunc("/version", infoHandler.GetVersion)
mux.HandleFunc("/docs", infoHandler.GetOpenAPIHTML)
mux.HandleFunc("/openapi.json", infoHandler.GetOpenAPIJSON)
```

### Probe Helpers

Readiness or liveness checks can be composed with the helper probes bundled in
`pkg/api/probe`. For MongoDB you can reuse the existing client connection:

```go
mongoProbe := probe.NewMongoPingProbe(mongoClient, nil)

infoHandler := info.NewInfoHandler(
  info.WithReadinessChecks(mongoProbe),
)
```

## Integration Notes

- The responder is transport agnostic and can be shared across handlers to keep
  error semantics consistent.
- Pair the info handler with the router package to get request validation and
  logging out of the box.
- When running behind a reverse proxy, set `WithBaseURL` so the HTML viewer
  fetches the specification from the correct origin.
- Use `WithOpenAPITemplate` and `WithOpenAPITemplateData` when you need to swap
  in a bespoke HTML shell or tweak the render-time context.

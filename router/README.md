# Router Package

## Overview

The `router` package produces an `http.ServeMux` preconfigured with validation,
logging, CORS, and timeout middleware. It is designed to sit in front of the
auto-generated handlers from `oapi-codegen`. A functional-options API keeps the
defaults lightweight while still allowing any middleware stack to be composed.

## Middleware Stack

Requests flow through the following middleware chain in order:

1. OpenAPI validation: requests are validated against the generated schema using
  `oapi-codegen` middleware.
2. CORS headers: driven by `CORSConfig` to make cross-origin calls predictable.
3. Timeout enforcement: ensures slow handlers do not occupy server resources
  indefinitely.
4. Structured logging: dumps method, path, headers (with optional redaction),
  and body size unless the route is listed in `QuietdownRoutes`.

## Configuration

- `Timeout`: per-request deadline applied by the HTTP timeout handler.
- `CORS`: lists allowed origins, methods, headers, and whether credentials are
  allowed.
- `QuietdownRoutes`: paths that should skip verbose request logging (useful for
  noisy health checks).
- `HideHeaders`: case-insensitive header keys that will be redacted before
  logging.

## Usage Example

```go
swagger, err := openapi3.NewLoader().LoadFromFile("./internal/server/_gen/openapi.json")
if err != nil {
    log.Fatal(err)
}

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
)

http.ListenAndServe(":8080", mux)
```

## Functional Options

- `router.WithConfig`, `router.WithConfigMutator`: supply or tweak the base
  `Config` without losing the defaults.
- `router.WithSwagger`, `router.WithLogger`: make the generated validation and
  logging middleware aware of the application's observability stack.
- `router.WithMiddlewares`, `router.WithTrailingMiddlewares`: inject custom
  middleware before or after the default stack.
- `router.WithMiddlewareChain`: fully replace the chain with a bespoke one.
- `router.Without*` helpers disable any default middleware you don't want.

Pair the router with the `api.InfoHandler` to expose health and documentation
endpoints on the same multiplexer.

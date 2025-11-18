// Package apiweaver bundles composable HTTP helpers for building
// well-instrumented Go services. The module stays intentionally small and
// encourages teams to pull in only the packages they need, keeping binaries
// lean and dependencies predictable.
//
// The responder package centralises JSON rendering, structured errors, and
// trace-friendly metadata. The info package layers on status, version, and
// documentation endpoints along with an HTML viewer for your OpenAPI document.
// Probe helpers make it trivial to wire health checks for databases or bespoke
// functions into readiness/liveness routes, while jsonutil provides thin sonic
// wrappers for high-throughput encoding and decoding.
//
// # Packages
//
//   - responder: consistent JSON success/error envelopes and structured logging
//     hooks via functional options.
//   - info: batteries-included health, version, and docs endpoints (including an
//     embedded OpenAPI viewer and static assets).
//   - probe: adapters that turn database ping functions or arbitrary closures
//     into HTTP-friendly readiness checks.
//   - jsonutil: tiny helpers around sonic for performance-sensitive encoding
//     tasks.
//
// # Quick Start
//
//	resp := responder.NewResponder(responder.WithLogger(logger))
//	infoHandler := info.NewInfoHandler(
//	    info.WithInfoResponder(resp),
//	    info.WithBaseURL("https://api.example.com"),
//	    info.WithReadinessChecks(probe.NewMongoPingProbe(mongoClient, nil)),
//	)
//
//	mux := http.NewServeMux()
//	mux.HandleFunc("/status", infoHandler.GetStatus)
//	mux.HandleFunc("/version", infoHandler.GetVersion)
//	mux.HandleFunc("/docs", infoHandler.GetOpenAPIHTML)
//
// Sharing the responder keeps JSON payloads, error envelopes, and trace IDs
// consistent. Additional options expose Swagger JSON, tweak the HTML template,
// or register extra readiness probes for bespoke dependencies.
package apiweaver

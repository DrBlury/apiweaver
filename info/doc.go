// Package info exposes build metadata, health probes, and OpenAPI endpoints.
//
// The package includes support for multiple OpenAPI documentation UIs:
//   - Stoplight Elements (default)
//   - Scalar
//   - SwaggerUI
//   - Redoc
//
// Use WithUIType to select your preferred UI when creating an InfoHandler.
//
// See ExampleInfoHandler_full for a runnable wiring of the handler and probes,
// and ExampleInfoHandler_differentUITypes for examples of using different UIs.
package info

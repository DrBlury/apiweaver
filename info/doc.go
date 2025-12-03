// Package info exposes build metadata, health probes, OpenAPI, and AsyncAPI endpoints.
//
// The package includes support for multiple OpenAPI documentation UIs:
//   - Stoplight Elements (default)
//   - Scalar
//   - SwaggerUI
//   - Redoc
//
// Use WithUIType to select your preferred UI when creating an InfoHandler.
//
// For event-driven APIs, the package also supports AsyncAPI documentation:
//   - Use WithAsyncAPIProvider to supply the AsyncAPI spec
//   - Use WithAsyncAPITemplate for custom HTML templates
//   - Use WithAsyncAPITemplateData for custom template data
//   - The default template uses AsyncAPI React Component
//
// See ExampleInfoHandler_full for a runnable wiring of the handler and probes,
// ExampleInfoHandler_differentUITypes for examples of using different OpenAPI UIs,
// and ExampleInfoHandler_asyncAPI for AsyncAPI documentation examples.
package info

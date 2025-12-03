package info

import (
	_ "embed"
	"html/template"
)

//go:embed assets/stoplight.html
var openapiHTMLStoplight []byte

//go:embed assets/scalar.html
var openapiHTMLScalar []byte

//go:embed assets/swaggerui.html
var openapiHTMLSwaggerUI []byte

//go:embed assets/redoc.html
var openapiHTMLRedoc []byte

//go:embed assets/asyncapi.html
var asyncapiHTML []byte

// UIType specifies which OpenAPI documentation UI to use.
type UIType string

const (
	// UIStoplight uses Stoplight Elements for OpenAPI rendering (default).
	UIStoplight UIType = "stoplight"
	// UIScalar uses Scalar for OpenAPI rendering.
	UIScalar UIType = "scalar"
	// UISwaggerUI uses SwaggerUI for OpenAPI rendering.
	UISwaggerUI UIType = "swaggerui"
	// UIRedoc uses Redoc for OpenAPI rendering.
	UIRedoc UIType = "redoc"
)

var (
	templateStoplight = template.Must(
		template.New("openapi-stoplight").Parse(string(openapiHTMLStoplight)),
	)
	templateScalar = template.Must(
		template.New("openapi-scalar").Parse(string(openapiHTMLScalar)),
	)
	templateSwaggerUI = template.Must(
		template.New("openapi-swaggerui").Parse(string(openapiHTMLSwaggerUI)),
	)
	templateRedoc = template.Must(
		template.New("openapi-redoc").Parse(string(openapiHTMLRedoc)),
	)
	templateAsyncAPI = template.Must(
		template.New("asyncapi").Parse(string(asyncapiHTML)),
	)
)

var defaultOpenAPITemplate = templateStoplight
var defaultAsyncAPITemplate = templateAsyncAPI

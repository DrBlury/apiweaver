package info

import (
	_ "embed"
	"html/template"
)

//go:embed assets/stoplight.html
var openapiHTMLStoplight []byte

var defaultOpenAPITemplate = template.Must(
	template.New("openapi-stoplight").Parse(string(openapiHTMLStoplight)),
)

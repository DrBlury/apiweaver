package info

import (
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/drblury/apiweaver/probe"
	"github.com/drblury/apiweaver/responder"
)

// InfoProvider returns the payload that will be exposed by the version endpoint.
// The provider allows callers to inject their own source for build metadata or
// runtime diagnostics.
type InfoProvider func() any

// SwaggerProvider returns the raw OpenAPI document that should be rendered by
// the documentation endpoints. It is commonly backed by an embedded JSON file
// generated at build time.
type SwaggerProvider func() ([]byte, error)

// InfoOption follows the functional options pattern used by NewInfoHandler to
// configure optional collaborators such as the responder, base URL, and
// information providers.
type InfoOption func(*InfoHandler)

// TemplateDataProvider allows callers to customise the data payload passed to
// the OpenAPI HTML template at render time.
type TemplateDataProvider func(r *http.Request, baseURL string) any

const defaultProbeTimeout = 2 * time.Second

// ProbeFunc is executed to determine the outcome of liveness or readiness
// probes. Returning a non-nil error marks the probe as failed.
type ProbeFunc = probe.Func

// InfoHandler wires the generated OpenAPI handlers with auxiliary endpoints
// that expose build information, status checks, and a pre-built HTML UI.
type InfoHandler struct {
	*responder.Responder
	baseURL         string
	infoProvider    InfoProvider
	swaggerProvider SwaggerProvider
	openapiTemplate *template.Template
	dataProvider    TemplateDataProvider
	probeTimeout    time.Duration
	livenessChecks  []ProbeFunc
	readinessChecks []ProbeFunc
	uiType          UIType
}

// NewInfoHandler constructs an InfoHandler with sensible defaults. Callers can
// supply InfoOption values to plug in domain specific providers or override the
// base responder implementation.
func NewInfoHandler(opts ...InfoOption) *InfoHandler {
	ih := &InfoHandler{
		Responder: responder.NewResponder(),
		infoProvider: func() any {
			return map[string]string{}
		},
		swaggerProvider: func() ([]byte, error) {
			return nil, errors.New("api swagger provider not configured")
		},
		openapiTemplate: defaultOpenAPITemplate,
		dataProvider:    defaultTemplateDataProvider,
		probeTimeout:    defaultProbeTimeout,
		uiType:          UIStoplight,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(ih)
		}
	}
	return ih
}

// WithInfoResponder replaces the responder used to craft JSON responses and
// handle error reporting.
func WithInfoResponder(responder *responder.Responder) InfoOption {
	return func(ih *InfoHandler) {
		if responder != nil {
			ih.Responder = responder
		}
	}
}

// WithBaseURL sets the URL prefix that will be injected into the rendered
// documentation template.
func WithBaseURL(baseURL string) InfoOption {
	return func(ih *InfoHandler) {
		ih.baseURL = baseURL
	}
}

// WithInfoProvider swaps the default metadata provider with a user supplied
// implementation.
func WithInfoProvider(provider InfoProvider) InfoOption {
	return func(ih *InfoHandler) {
		if provider != nil {
			ih.infoProvider = provider
		}
	}
}

// WithSwaggerProvider sets the source of the OpenAPI JSON document that backs
// the documentation endpoints.
func WithSwaggerProvider(provider SwaggerProvider) InfoOption {
	return func(ih *InfoHandler) {
		if provider != nil {
			ih.swaggerProvider = provider
		}
	}
}

// WithOpenAPITemplate injects a custom html/template instance used to render
// the OpenAPI viewer page. Callers can parse templates from disk or embed them
// via go:embed before passing them in.
func WithOpenAPITemplate(tmpl *template.Template) InfoOption {
	return func(ih *InfoHandler) {
		if tmpl != nil {
			ih.openapiTemplate = tmpl
		}
	}
}

// WithOpenAPITemplateData overrides the template data provider that runs for
// each request to the HTML endpoint.
func WithOpenAPITemplateData(provider TemplateDataProvider) InfoOption {
	return func(ih *InfoHandler) {
		if provider != nil {
			ih.dataProvider = provider
		}
	}
}

// WithProbeTimeout adjusts the maximum duration allowed for probe checks.
func WithProbeTimeout(timeout time.Duration) InfoOption {
	return func(ih *InfoHandler) {
		if timeout > 0 {
			ih.probeTimeout = timeout
		}
	}
}

// WithLivenessChecks replaces the default liveness checks with the supplied
// functions.
func WithLivenessChecks(checks ...ProbeFunc) InfoOption {
	return func(ih *InfoHandler) {
		ih.livenessChecks = filterProbes(checks)
	}
}

// WithReadinessChecks replaces the default readiness checks with the supplied
// functions.
func WithReadinessChecks(checks ...ProbeFunc) InfoOption {
	return func(ih *InfoHandler) {
		ih.readinessChecks = filterProbes(checks)
	}
}

// WithUIType sets the OpenAPI documentation UI to use. Supported values are
// UIStoplight (default), UIScalar, UISwaggerUI, and UIRedoc.
func WithUIType(uiType UIType) InfoOption {
	return func(ih *InfoHandler) {
		ih.uiType = uiType
		switch uiType {
		case UIScalar:
			ih.openapiTemplate = templateScalar
		case UISwaggerUI:
			ih.openapiTemplate = templateSwaggerUI
		case UIRedoc:
			ih.openapiTemplate = templateRedoc
		case UIStoplight:
			ih.openapiTemplate = templateStoplight
		default:
			ih.openapiTemplate = templateStoplight
		}
	}
}

func defaultTemplateDataProvider(_ *http.Request, baseURL string) any {
	return map[string]any{
		"BaseURL": baseURL,
	}
}

package info_test

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/drblury/apiweaver/info"
	"github.com/drblury/apiweaver/probe"
)

func ExampleInfoHandler_full() {
	handler := info.NewInfoHandler(
		info.WithBaseURL("/docs"),
		info.WithInfoProvider(func() any {
			return map[string]string{"version": "1.2.3"}
		}),
		info.WithSwaggerProvider(func() ([]byte, error) {
			return []byte(`{"openapi":"3.1.0","info":{"title":"Demo","version":"1.0.0"}}`), nil
		}),
		info.WithLivenessChecks(probe.NewPingProbe("noop", func(ctx context.Context) error {
			return nil
		})),
		info.WithReadinessChecks(probe.NewPingProbe("db", func(ctx context.Context) error {
			return nil
		})),
	)

	healthRec := httptest.NewRecorder()
	handler.GetHealthz(healthRec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	fmt.Println(healthRec.Code)
	fmt.Println(strings.TrimSpace(healthRec.Body.String()))

	versionRec := httptest.NewRecorder()
	handler.GetVersion(versionRec, httptest.NewRequest(http.MethodGet, "/version", nil))
	fmt.Println(versionRec.Code)
	fmt.Println(strings.TrimSpace(versionRec.Body.String()))

	// Output:
	// 200
	// {"status":"ok"}
	// 200
	// {"version":"1.2.3"}
}

func ExampleInfoHandler_customTemplate() {
	handler := info.NewInfoHandler(
		info.WithBaseURL("https://api.example.com"),
		info.WithOpenAPITemplate(template.Must(template.New("docs").Parse(`<div>{{.DocsURL}}</div>`))),
		info.WithOpenAPITemplateData(func(r *http.Request, baseURL string) any {
			return map[string]string{
				"DocsURL": baseURL + "/info/openapi.json",
			}
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()
	handler.GetOpenAPIHTML(rr, req)

	fmt.Println(rr.Code)
	fmt.Println(strings.TrimSpace(rr.Body.String()))
	// Output:
	// 200
	// <div>https://api.example.com/info/openapi.json</div>
}

func ExampleInfoHandler_differentUITypes() {
	// Create handlers with different UI types
	handlerScalar := info.NewInfoHandler(
		info.WithBaseURL("https://api.example.com"),
		info.WithUIType(info.UIScalar),
		info.WithSwaggerProvider(func() ([]byte, error) {
			return []byte(`{"openapi":"3.1.0"}`), nil
		}),
	)

	handlerSwaggerUI := info.NewInfoHandler(
		info.WithBaseURL("https://api.example.com"),
		info.WithUIType(info.UISwaggerUI),
	)

	handlerRedoc := info.NewInfoHandler(
		info.WithBaseURL("https://api.example.com"),
		info.WithUIType(info.UIRedoc),
	)

	// Test Scalar UI
	reqScalar := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rrScalar := httptest.NewRecorder()
	handlerScalar.GetOpenAPIHTML(rrScalar, reqScalar)
	fmt.Println("Scalar:", rrScalar.Code, strings.Contains(rrScalar.Body.String(), "@scalar/api-reference"))

	// Test SwaggerUI
	reqSwagger := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rrSwagger := httptest.NewRecorder()
	handlerSwaggerUI.GetOpenAPIHTML(rrSwagger, reqSwagger)
	fmt.Println("SwaggerUI:", rrSwagger.Code, strings.Contains(rrSwagger.Body.String(), "swagger-ui-dist"))

	// Test Redoc
	reqRedoc := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rrRedoc := httptest.NewRecorder()
	handlerRedoc.GetOpenAPIHTML(rrRedoc, reqRedoc)
	fmt.Println("Redoc:", rrRedoc.Code, strings.Contains(rrRedoc.Body.String(), "redoc"))

	// Output:
	// Scalar: 200 true
	// SwaggerUI: 200 true
	// Redoc: 200 true
}

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

func ExampleInfoHandler_asyncAPI() {
	handler := info.NewInfoHandler(
		info.WithBaseURL("https://events.example.com"),
		info.WithAsyncAPIProvider(func() ([]byte, error) {
			return []byte(`{"asyncapi":"3.0.0","info":{"title":"Events API","version":"1.0.0"}}`), nil
		}),
	)

	// Test AsyncAPI JSON endpoint
	jsonRec := httptest.NewRecorder()
	handler.GetAsyncAPIJSON(jsonRec, httptest.NewRequest(http.MethodGet, "/info/asyncapi.json", nil))
	fmt.Println("JSON:", jsonRec.Code)
	fmt.Println(strings.TrimSpace(jsonRec.Body.String()))

	// Test AsyncAPI HTML endpoint
	htmlRec := httptest.NewRecorder()
	handler.GetAsyncAPIHTML(htmlRec, httptest.NewRequest(http.MethodGet, "/info/asyncapi.html", nil))
	fmt.Println("HTML:", htmlRec.Code, strings.Contains(htmlRec.Body.String(), "@asyncapi/react-component"))

	// Use the helper function to get the spec URL
	fmt.Println("Spec URL:", info.AsyncAPISpecURL("https://events.example.com"))

	// Output:
	// JSON: 200
	// {"asyncapi":"3.0.0","info":{"title":"Events API","version":"1.0.0"}}
	// HTML: 200 true
	// Spec URL: https://events.example.com/info/asyncapi.json
}

func ExampleInfoHandler_asyncAPICustomTemplate() {
	handler := info.NewInfoHandler(
		info.WithBaseURL("https://events.example.com"),
		info.WithAsyncAPITemplate(template.Must(template.New("events").Parse(`<div>{{.EventsURL}}</div>`))),
		info.WithAsyncAPITemplateData(func(r *http.Request, baseURL string) any {
			return map[string]string{
				"EventsURL": baseURL + "/info/asyncapi.json",
			}
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()
	handler.GetAsyncAPIHTML(rr, req)

	fmt.Println(rr.Code)
	fmt.Println(strings.TrimSpace(rr.Body.String()))
	// Output:
	// 200
	// <div>https://events.example.com/info/asyncapi.json</div>
}

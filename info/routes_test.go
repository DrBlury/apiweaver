package info

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInfoHandler_GetStatus(t *testing.T) {
	handler := NewInfoHandler()
	req := httptest.NewRequest(http.MethodGet, "/status", nil)
	rr := httptest.NewRecorder()

	handler.GetStatus(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}

	payload := decodeProbePayload(t, rr.Body.Bytes())
	if payload.Status != "HEALTHY" {
		t.Fatalf("expected status HEALTHY, got %s", payload.Status)
	}
}

func TestInfoHandler_GetHealthz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewInfoHandler(WithLivenessChecks(func(context.Context) error { return nil }))
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		handler.GetHealthz(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		payload := decodeProbePayload(t, rr.Body.Bytes())
		if payload.Status != "ok" {
			t.Fatalf("expected status ok, got %s", payload.Status)
		}
	})

	t.Run("failure propagates probe error", func(t *testing.T) {
		sentinel := errors.New("db down")
		handler := NewInfoHandler(WithLivenessChecks(func(context.Context) error { return sentinel }))
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()

		handler.GetHealthz(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}

		problem := decodeProblemDetails(t, rr.Body.Bytes())
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("expected problem status %d, got %d", http.StatusServiceUnavailable, problem.Status)
		}
		if !strings.Contains(problem.Detail, sentinel.Error()) {
			t.Fatalf("expected detail to include %q, got %q", sentinel.Error(), problem.Detail)
		}
	})
}

func TestInfoHandler_GetReadyz(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		handler := NewInfoHandler(WithReadinessChecks(func(context.Context) error { return nil }))
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()

		handler.GetReadyz(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		payload := decodeProbePayload(t, rr.Body.Bytes())
		if payload.Status != "ready" {
			t.Fatalf("expected status ready, got %s", payload.Status)
		}
	})

	t.Run("failure propagates probe error", func(t *testing.T) {
		sentinel := errors.New("cache warming")
		handler := NewInfoHandler(WithReadinessChecks(func(context.Context) error { return sentinel }))
		req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
		rr := httptest.NewRecorder()

		handler.GetReadyz(rr, req)

		if rr.Code != http.StatusServiceUnavailable {
			t.Fatalf("expected status %d, got %d", http.StatusServiceUnavailable, rr.Code)
		}

		problem := decodeProblemDetails(t, rr.Body.Bytes())
		if problem.Status != http.StatusServiceUnavailable {
			t.Fatalf("expected problem status %d, got %d", http.StatusServiceUnavailable, problem.Status)
		}
		if !strings.Contains(problem.Detail, sentinel.Error()) {
			t.Fatalf("expected detail to include %q, got %q", sentinel.Error(), problem.Detail)
		}
	})
}

func TestInfoHandler_GetVersion(t *testing.T) {
	t.Run("uses configured provider", func(t *testing.T) {
		handler := NewInfoHandler(WithInfoProvider(func() any {
			return map[string]string{"commit": "abc123"}
		}))
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		rr := httptest.NewRecorder()

		handler.GetVersion(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode version payload: %v", err)
		}
		if payload["commit"] != "abc123" {
			t.Fatalf("expected commit abc123, got %s", payload["commit"])
		}
	})

	t.Run("falls back to empty map when provider returns nil", func(t *testing.T) {
		handler := NewInfoHandler(WithInfoProvider(func() any { return nil }))
		req := httptest.NewRequest(http.MethodGet, "/version", nil)
		rr := httptest.NewRecorder()

		handler.GetVersion(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var payload map[string]string
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("failed to decode version payload: %v", err)
		}
		if len(payload) != 0 {
			t.Fatalf("expected empty payload, got %v", payload)
		}
	})
}

func TestInfoHandler_GetOpenAPIJSON(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := []byte(`{"openapi":"3.0.0"}`)
		handler := NewInfoHandler(WithSwaggerProvider(func() ([]byte, error) {
			return expected, nil
		}))
		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		rr := httptest.NewRecorder()

		handler.GetOpenAPIJSON(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %s", got)
		}
		if !bytes.Equal(rr.Body.Bytes(), expected) {
			t.Fatalf("expected body %s, got %s", expected, rr.Body.Bytes())
		}
	})

	t.Run("provider error is surfaced", func(t *testing.T) {
		sentinel := errors.New("missing spec")
		handler := NewInfoHandler(WithSwaggerProvider(func() ([]byte, error) {
			return nil, sentinel
		}))
		req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
		rr := httptest.NewRecorder()

		handler.GetOpenAPIJSON(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		problem := decodeProblemDetails(t, rr.Body.Bytes())
		if problem.Status != http.StatusInternalServerError {
			t.Fatalf("expected problem status %d, got %d", http.StatusInternalServerError, problem.Status)
		}
		if !strings.Contains(problem.Detail, sentinel.Error()) {
			t.Fatalf("expected detail to include %q, got %q", sentinel.Error(), problem.Detail)
		}
	})
}

func TestInfoHandler_GetOpenAPIHTML(t *testing.T) {
	t.Run("renders template with custom data", testGetOpenAPIHTMLCustomData)
	t.Run("falls back to default data provider", testGetOpenAPIHTMLDefaultData)
	t.Run("missing template returns problem response", testGetOpenAPIHTMLMissingTemplate)
	t.Run("template execution errors are surfaced", testGetOpenAPIHTMLTemplateError)
	t.Run("renders different UI types", testGetOpenAPIHTMLDifferentUITypes)
}

func testGetOpenAPIHTMLCustomData(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Parse(`{{.BaseURL}}|{{.Value}}`))
	called := false
	handler := NewInfoHandler(
		WithBaseURL("https://docs.example"),
		WithOpenAPITemplate(tmpl),
		WithOpenAPITemplateData(func(r *http.Request, base string) any {
			called = true
			if base != "https://docs.example" {
				t.Fatalf("expected base URL to be forwarded")
			}
			return map[string]string{
				"BaseURL": base,
				"Value":   "custom",
			}
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	handler.GetOpenAPIHTML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Header().Get("Content-Type") != "text/html" {
		t.Fatalf("expected text/html content type, got %s", rr.Header().Get("Content-Type"))
	}
	if !called {
		t.Fatal("expected template data provider to be called")
	}
	if strings.TrimSpace(rr.Body.String()) != "https://docs.example|custom" {
		t.Fatalf("unexpected body %q", rr.Body.String())
	}
}

func testGetOpenAPIHTMLDefaultData(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Parse(`{{.BaseURL}}`))
	handler := NewInfoHandler(
		WithBaseURL("https://fallback"),
		WithOpenAPITemplate(tmpl),
		WithOpenAPITemplateData(func(*http.Request, string) any { return nil }),
	)
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	handler.GetOpenAPIHTML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "https://fallback" {
		t.Fatalf("expected body to use base url, got %q", rr.Body.String())
	}
}

func testGetOpenAPIHTMLMissingTemplate(t *testing.T) {
	t.Helper()
	handler := NewInfoHandler()
	handler.openapiTemplate = nil
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	handler.GetOpenAPIHTML(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	problem := decodeProblemDetails(t, rr.Body.Bytes())
	if !strings.Contains(problem.Detail, "openapi template not configured") {
		t.Fatalf("unexpected problem detail %q", problem.Detail)
	}
}

func testGetOpenAPIHTMLTemplateError(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Funcs(template.FuncMap{
		"boom": func() (string, error) {
			return "", errors.New("render failure")
		},
	}).Parse(`{{boom}}`))
	handler := NewInfoHandler(WithOpenAPITemplate(tmpl))
	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rr := httptest.NewRecorder()

	handler.GetOpenAPIHTML(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	problem := decodeProblemDetails(t, rr.Body.Bytes())
	if !strings.Contains(problem.Detail, "render failure") {
		t.Fatalf("expected detail to include render failure, got %q", problem.Detail)
	}
}

func testGetOpenAPIHTMLDifferentUITypes(t *testing.T) {
	t.Helper()
	testCases := []struct {
		name           string
		uiType         UIType
		expectedInBody string
	}{
		{
			name:           "stoplight UI",
			uiType:         UIStoplight,
			expectedInBody: "@stoplight/elements",
		},
		{
			name:           "scalar UI",
			uiType:         UIScalar,
			expectedInBody: "@scalar/api-reference",
		},
		{
			name:           "swaggerui UI",
			uiType:         UISwaggerUI,
			expectedInBody: "swagger-ui-dist",
		},
		{
			name:           "redoc UI",
			uiType:         UIRedoc,
			expectedInBody: "redoc",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewInfoHandler(
				WithBaseURL("https://api.example.com"),
				WithUIType(tc.uiType),
			)
			req := httptest.NewRequest(http.MethodGet, "/docs", nil)
			rr := httptest.NewRecorder()

			handler.GetOpenAPIHTML(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
			}
			if rr.Header().Get("Content-Type") != "text/html" {
				t.Fatalf("expected text/html content type, got %s", rr.Header().Get("Content-Type"))
			}
			body := rr.Body.String()
			if !strings.Contains(body, tc.expectedInBody) {
				t.Fatalf("expected body to contain %q, but got: %s", tc.expectedInBody, body)
			}
			// Verify BaseURL is rendered in all templates (may be escaped in JSON)
			if !strings.Contains(body, "https://api.example.com") && !strings.Contains(body, "https:\\/\\/api.example.com") {
				t.Fatalf("expected body to contain BaseURL, but got: %s", body)
			}
		})
	}
}

func TestInfoHandler_GetAsyncAPIJSON(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		expected := []byte(`{"asyncapi":"3.0.0"}`)
		handler := NewInfoHandler(WithAsyncAPIProvider(func() ([]byte, error) {
			return expected, nil
		}))
		req := httptest.NewRequest(http.MethodGet, "/asyncapi.json", nil)
		rr := httptest.NewRecorder()

		handler.GetAsyncAPIJSON(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
		if got := rr.Header().Get("Content-Type"); got != "application/json" {
			t.Fatalf("expected Content-Type application/json, got %s", got)
		}
		if !bytes.Equal(rr.Body.Bytes(), expected) {
			t.Fatalf("expected body %s, got %s", expected, rr.Body.Bytes())
		}
	})

	t.Run("provider error is surfaced", func(t *testing.T) {
		sentinel := errors.New("missing asyncapi spec")
		handler := NewInfoHandler(WithAsyncAPIProvider(func() ([]byte, error) {
			return nil, sentinel
		}))
		req := httptest.NewRequest(http.MethodGet, "/asyncapi.json", nil)
		rr := httptest.NewRecorder()

		handler.GetAsyncAPIJSON(rr, req)

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
		}
		problem := decodeProblemDetails(t, rr.Body.Bytes())
		if problem.Status != http.StatusInternalServerError {
			t.Fatalf("expected problem status %d, got %d", http.StatusInternalServerError, problem.Status)
		}
		if !strings.Contains(problem.Detail, sentinel.Error()) {
			t.Fatalf("expected detail to include %q, got %q", sentinel.Error(), problem.Detail)
		}
	})
}

func TestInfoHandler_GetAsyncAPIHTML(t *testing.T) {
	t.Run("renders template with custom data", testGetAsyncAPIHTMLCustomData)
	t.Run("falls back to default data provider", testGetAsyncAPIHTMLDefaultData)
	t.Run("missing template returns problem response", testGetAsyncAPIHTMLMissingTemplate)
	t.Run("template execution errors are surfaced", testGetAsyncAPIHTMLTemplateError)
	t.Run("renders default asyncapi template", testGetAsyncAPIHTMLDefaultTemplate)
}

func testGetAsyncAPIHTMLCustomData(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Parse(`{{.BaseURL}}|{{.Value}}`))
	called := false
	handler := NewInfoHandler(
		WithBaseURL("https://events.example"),
		WithAsyncAPITemplate(tmpl),
		WithAsyncAPITemplateData(func(r *http.Request, base string) any {
			called = true
			if base != "https://events.example" {
				t.Fatalf("expected base URL to be forwarded")
			}
			return map[string]string{
				"BaseURL": base,
				"Value":   "custom",
			}
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()

	handler.GetAsyncAPIHTML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Header().Get("Content-Type") != "text/html" {
		t.Fatalf("expected text/html content type, got %s", rr.Header().Get("Content-Type"))
	}
	if !called {
		t.Fatal("expected template data provider to be called")
	}
	if strings.TrimSpace(rr.Body.String()) != "https://events.example|custom" {
		t.Fatalf("unexpected body %q", rr.Body.String())
	}
}

func testGetAsyncAPIHTMLDefaultData(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Parse(`{{.BaseURL}}`))
	handler := NewInfoHandler(
		WithBaseURL("https://fallback"),
		WithAsyncAPITemplate(tmpl),
		WithAsyncAPITemplateData(func(*http.Request, string) any { return nil }),
	)
	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()

	handler.GetAsyncAPIHTML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if strings.TrimSpace(rr.Body.String()) != "https://fallback" {
		t.Fatalf("expected body to use base url, got %q", rr.Body.String())
	}
}

func testGetAsyncAPIHTMLMissingTemplate(t *testing.T) {
	t.Helper()
	handler := NewInfoHandler()
	handler.asyncapiTemplate = nil
	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()

	handler.GetAsyncAPIHTML(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	problem := decodeProblemDetails(t, rr.Body.Bytes())
	if !strings.Contains(problem.Detail, "asyncapi template not configured") {
		t.Fatalf("unexpected problem detail %q", problem.Detail)
	}
}

func testGetAsyncAPIHTMLTemplateError(t *testing.T) {
	t.Helper()
	tmpl := template.Must(template.New("test").Funcs(template.FuncMap{
		"boom": func() (string, error) {
			return "", errors.New("render failure")
		},
	}).Parse(`{{boom}}`))
	handler := NewInfoHandler(WithAsyncAPITemplate(tmpl))
	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()

	handler.GetAsyncAPIHTML(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rr.Code)
	}
	problem := decodeProblemDetails(t, rr.Body.Bytes())
	if !strings.Contains(problem.Detail, "render failure") {
		t.Fatalf("expected detail to include render failure, got %q", problem.Detail)
	}
}

func testGetAsyncAPIHTMLDefaultTemplate(t *testing.T) {
	t.Helper()
	handler := NewInfoHandler(
		WithBaseURL("https://api.example.com"),
		WithAsyncAPIProvider(func() ([]byte, error) {
			return []byte(`{"asyncapi":"3.0.0"}`), nil
		}),
	)
	req := httptest.NewRequest(http.MethodGet, "/asyncapi", nil)
	rr := httptest.NewRecorder()

	handler.GetAsyncAPIHTML(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rr.Code)
	}
	if rr.Header().Get("Content-Type") != "text/html" {
		t.Fatalf("expected text/html content type, got %s", rr.Header().Get("Content-Type"))
	}
	body := rr.Body.String()
	// Verify the default template contains AsyncAPI React Component
	if !strings.Contains(body, "@asyncapi/react-component") {
		t.Fatalf("expected body to contain @asyncapi/react-component, but got: %s", body)
	}
	// Verify BaseURL is rendered (may be escaped in JavaScript)
	if !strings.Contains(body, "https://api.example.com") && !strings.Contains(body, "https:\\/\\/api.example.com") {
		t.Fatalf("expected body to contain BaseURL, but got: %s", body)
	}
}

func TestAsyncAPISpecURL(t *testing.T) {
	testCases := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "simple base URL",
			baseURL:  "https://api.example.com",
			expected: "https://api.example.com/info/asyncapi.json",
		},
		{
			name:     "base URL with path",
			baseURL:  "https://api.example.com/v1",
			expected: "https://api.example.com/v1/info/asyncapi.json",
		},
		{
			name:     "empty base URL",
			baseURL:  "",
			expected: "/info/asyncapi.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := AsyncAPISpecURL(tc.baseURL)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

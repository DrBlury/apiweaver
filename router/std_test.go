package router

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestNewAllowsMiddlewareOverride(t *testing.T) {
	var order []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusTeapot)
	})

	mux := New(handler, WithMiddlewareChain(
		recordingMiddleware("one", &order),
		recordingMiddleware("two", &order),
	))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	expected := []string{"one-before", "two-before", "handler", "two-after", "one-after"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("unexpected middleware order: got %v, want %v", order, expected)
	}

	if rr.Code != http.StatusTeapot {
		t.Fatalf("unexpected response code: got %d want %d", rr.Code, http.StatusTeapot)
	}
}

func TestNewSupportsPrependAndAppendMiddlewares(t *testing.T) {
	var order []string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusNoContent)
	})

	mux := New(
		handler,
		WithoutOpenAPIValidation(),
		WithoutCORSMiddleware(),
		WithoutTimeoutMiddleware(),
		WithoutLoggingMiddleware(),
		WithMiddlewares(recordingMiddleware("outer", &order)),
		WithTrailingMiddlewares(recordingMiddleware("inner", &order)),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	expected := []string{"outer-before", "inner-before", "handler", "inner-after", "outer-after"}
	if !reflect.DeepEqual(order, expected) {
		t.Fatalf("unexpected middleware order: got %v want %v", order, expected)
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected response code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestNewAppliesCORSEnforcementFromConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux := New(
		handler,
		WithConfigMutator(func(cfg *Config) {
			cfg.CORS = CORSConfig{
				Origins:          []string{"https://example.com"},
				Methods:          []string{http.MethodGet, http.MethodPost},
				Headers:          []string{"Content-Type"},
				AllowCredentials: true,
			}
		}),
	)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusOK)
	}

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Fatalf("unexpected access-control-allow-origin: got %q want %q", got, "https://example.com")
	}

	if got := rr.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST" {
		t.Fatalf("unexpected access-control-allow-methods: got %q want %q", got, "GET,POST")
	}

	if got := rr.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type" {
		t.Fatalf("unexpected access-control-allow-headers: got %q want %q", got, "Content-Type")
	}

	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("unexpected access-control-allow-credentials: got %q want %q", got, "true")
	}
}

func TestWithoutCORSMiddlewareSkipsHeaders(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	mux := New(
		handler,
		WithConfigMutator(func(cfg *Config) {
			cfg.CORS = CORSConfig{
				Origins: []string{"https://example.com"},
				Methods: []string{http.MethodGet},
				Headers: []string{"Authorization"},
			}
		}),
		WithoutCORSMiddleware(),
	)

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("expected CORS headers to be skipped when middleware disabled")
	}

	if rr.Code != http.StatusNoContent {
		t.Fatalf("unexpected status code: got %d want %d", rr.Code, http.StatusNoContent)
	}
}

func TestTimeoutMiddlewareCanBeDisabled(t *testing.T) {
	longHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	withTimeout := New(
		longHandler,
		WithConfig(Config{Timeout: 1 * time.Millisecond}),
	)

	withoutTimeout := New(
		longHandler,
		WithConfig(Config{Timeout: 1 * time.Millisecond}),
		WithoutTimeoutMiddleware(),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)

	rrTimeout := httptest.NewRecorder()
	withTimeout.ServeHTTP(rrTimeout, req)
	if rrTimeout.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected timeout handler to fire, got %d", rrTimeout.Code)
	}

	rrNoTimeout := httptest.NewRecorder()
	withoutTimeout.ServeHTTP(rrNoTimeout, req)
	if rrNoTimeout.Code != http.StatusOK {
		t.Fatalf("expected handler to complete when timeout disabled, got %d", rrNoTimeout.Code)
	}
}

func TestNewPanicsWhenHandlerNil(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when handler is nil")
		}
	}()

	New(nil)
}

func recordingMiddleware(label string, sink *[]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*sink = append(*sink, label+"-before")
			next.ServeHTTP(w, r)
			*sink = append(*sink, label+"-after")
		})
	}
}

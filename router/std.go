package router

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	oapiMW "github.com/oapi-codegen/nethttp-middleware"
)

// New returns a new *http.ServeMux configured with the provided handler and options.
func New(apiHandle http.Handler, opts ...Option) *http.ServeMux {
	if apiHandle == nil {
		panic("router: handler cannot be nil")
	}

	settings := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(settings)
		}
	}

	finalHandler := applyMiddlewares(apiHandle, settings.middlewareChain())
	mux := http.NewServeMux()
	mux.Handle("/", finalHandler)
	return mux
}

func applyMiddlewares(handler http.Handler, middlewares []Middleware) http.Handler {
	if len(middlewares) == 0 {
		return handler
	}

	for i := len(middlewares) - 1; i >= 0; i-- {
		middleware := middlewares[i]
		if middleware == nil {
			continue
		}
		handler = middleware(handler)
	}

	return handler
}

func oapiMiddleware(swagger *openapi3.T) Middleware {
	return func(next http.Handler) http.Handler {
		// Clear out the servers array in the swagger spec, that skips validating
		// that server names match. We don't know how this thing will be run.
		swagger.Servers = nil

		// Validate requests against OpenAPI spec
		validatorOptions := &oapiMW.Options{
			Options: openapi3filter.Options{
				AuthenticationFunc: func(c context.Context, input *openapi3filter.AuthenticationInput) error {
					return nil
				},
			},
		}

		return oapiMW.OapiRequestValidatorWithOptions(swagger, validatorOptions)(next)
	}
}

func loggingMiddleware(logger *slog.Logger, quietdownRoutes []string, hideHeaders []string) Middleware {
	logger.With(
		"QuietdownRoutes", quietdownRoutes,
		"HideHeaders", hideHeaders,
	).Debug("Config for logging middleware")

	quietRoutesCopy := cloneStrings(quietdownRoutes)
	redactedCopy := cloneStrings(hideHeaders)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !shouldQuietRoute(r.URL.Path, quietRoutesCopy) {
				headers := cloneHeaders(r.Header)
				redactHeaders(headers, redactedCopy)

				attrs := []any{
					"Path", r.URL.Path,
					"Method", r.Method,
					"Header", headers,
				}

				if r.ContentLength > 0 {
					attrs = append(attrs, "ContentLength", r.ContentLength)
				}

				logger.With(attrs...).Debug("Request")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware adds CORS headers based on the provided configuration.
func corsMiddleware(cfg CORSConfig) Middleware {
	headersCopy := cloneStrings(cfg.Headers)
	methodsCopy := cloneStrings(cfg.Methods)
	originsCopy := cloneStrings(cfg.Origins)

	return func(next http.Handler) http.Handler {
		if len(originsCopy) == 0 {
			return next
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			if allowedOrigin(origin, originsCopy) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(methodsCopy, ","))
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(headersCopy, ","))
				if cfg.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// timeoutMiddleware adds timeout handling to requests.
func timeoutMiddleware(timeout time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, "Timeout")
	}
}

func allowedOrigin(origin string, allowed []string) bool {
	for _, candidate := range allowed {
		if candidate == "*" || candidate == origin {
			return true
		}
	}

	return false
}

func shouldQuietRoute(path string, quietdownRoutes []string) bool {
	for _, quietPath := range quietdownRoutes {
		if path == quietPath {
			return true
		}
	}

	return false
}

func cloneHeaders(src http.Header) http.Header {
	headers := make(http.Header, len(src))
	for k, v := range src {
		copied := make([]string, len(v))
		copy(copied, v)
		headers[k] = copied
	}

	return headers
}

func redactHeaders(headers http.Header, hideHeaders []string) {
	for _, header := range hideHeaders {
		canonical := http.CanonicalHeaderKey(header)
		values, exists := headers[canonical]
		if !exists {
			continue
		}

		redactedLen := 0
		for _, value := range values {
			redactedLen += len(value)
		}

		headers[canonical] = []string{fmt.Sprintf("[REDACTED - %d bytes]", redactedLen)}
	}
}

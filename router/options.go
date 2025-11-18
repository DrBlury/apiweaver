package router

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
)

// Middleware wraps an http.Handler to produce a new http.Handler.
type Middleware func(http.Handler) http.Handler

// Option configures the router via the functional options pattern.
type Option func(*options)

type options struct {
	config        Config
	logger        *slog.Logger
	swagger       *openapi3.T
	prepend       []Middleware
	append        []Middleware
	override      []Middleware
	enableOpenAPI bool
	enableCORS    bool
	enableTimeout bool
	enableLogging bool
}

func defaultOptions() *options {
	return &options{
		config: Config{
			Timeout: 30 * time.Second,
		},
		logger:        slog.Default(),
		enableOpenAPI: true,
		enableCORS:    true,
		enableTimeout: true,
		enableLogging: true,
	}
}

func (o *options) middlewareChain() []Middleware {
	if len(o.override) > 0 {
		cloned := make([]Middleware, len(o.override))
		copy(cloned, o.override)
		return cloned
	}

	chain := make([]Middleware, 0, len(o.prepend)+len(o.append)+4)
	chain = append(chain, o.prepend...)
	chain = append(chain, o.defaultMiddlewares()...)
	chain = append(chain, o.append...)
	return chain
}

func (o *options) defaultMiddlewares() []Middleware {
	chain := make([]Middleware, 0, 4)

	if o.enableOpenAPI && o.swagger != nil {
		chain = append(chain, oapiMiddleware(o.swagger))
	}

	if o.enableCORS && shouldApplyCORS(o.config.CORS) {
		chain = append(chain, corsMiddleware(o.config.CORS))
	}

	if o.enableTimeout && o.config.Timeout > 0 {
		chain = append(chain, timeoutMiddleware(o.config.Timeout))
	}

	if o.enableLogging && o.logger != nil {
		chain = append(chain, loggingMiddleware(o.logger, o.config.QuietdownRoutes, o.config.HideHeaders))
	}

	return chain
}

// WithConfig replaces the router configuration with the provided value.
func WithConfig(cfg Config) Option {
	configCopy := sanitizeConfig(cfg)
	return func(o *options) {
		o.config = configCopy
	}
}

// WithConfigMutator applies a mutation to the router configuration after defaults are set.
func WithConfigMutator(mutator func(*Config)) Option {
	return func(o *options) {
		if mutator != nil {
			mutator(&o.config)
		}
	}
}

// WithLogger provides the structured logger to be used by the logging middleware.
func WithLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// WithSwagger wires the OpenAPI document for request validation.
func WithSwagger(swagger *openapi3.T) Option {
	return func(o *options) {
		o.swagger = swagger
	}
}

// WithMiddlewares prepends custom middlewares ahead of the default chain.
func WithMiddlewares(middlewares ...Middleware) Option {
	return func(o *options) {
		o.prepend = append(o.prepend, middlewares...)
	}
}

// WithTrailingMiddlewares appends middlewares after the default chain.
func WithTrailingMiddlewares(middlewares ...Middleware) Option {
	return func(o *options) {
		o.append = append(o.append, middlewares...)
	}
}

// WithMiddlewareChain fully overrides the middleware chain with the provided sequence.
func WithMiddlewareChain(middlewares ...Middleware) Option {
	cloned := make([]Middleware, len(middlewares))
	copy(cloned, middlewares)
	return func(o *options) {
		o.override = cloned
	}
}

// WithoutOpenAPIValidation disables the OpenAPI validation middleware.
func WithoutOpenAPIValidation() Option {
	return func(o *options) {
		o.enableOpenAPI = false
	}
}

// WithoutCORSMiddleware disables the CORS middleware regardless of configuration.
func WithoutCORSMiddleware() Option {
	return func(o *options) {
		o.enableCORS = false
	}
}

// WithoutTimeoutMiddleware disables the timeout middleware.
func WithoutTimeoutMiddleware() Option {
	return func(o *options) {
		o.enableTimeout = false
	}
}

// WithoutLoggingMiddleware disables the logging middleware.
func WithoutLoggingMiddleware() Option {
	return func(o *options) {
		o.enableLogging = false
	}
}

func sanitizeConfig(cfg Config) Config {
	cfg.QuietdownRoutes = cloneStrings(cfg.QuietdownRoutes)
	cfg.HideHeaders = cloneStrings(cfg.HideHeaders)
	cfg.CORS = sanitizeCORSConfig(cfg.CORS)
	return cfg
}

func sanitizeCORSConfig(cfg CORSConfig) CORSConfig {
	cfg.Headers = cloneStrings(cfg.Headers)
	cfg.Methods = cloneStrings(cfg.Methods)
	cfg.Origins = cloneStrings(cfg.Origins)
	return cfg
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	cloned := make([]string, len(values))
	copy(cloned, values)
	return cloned
}

func shouldApplyCORS(cfg CORSConfig) bool {
	return len(cfg.Origins) > 0
}

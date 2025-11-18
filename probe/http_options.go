package probe

import (
	"fmt"
	"net/http"
)

// HTTPStatusExpectation determines whether a given HTTP status code is acceptable.
type HTTPStatusExpectation func(status int) bool

// HTTPRequestMutator allows callers to tweak the outbound request prior to dispatch.
type HTTPRequestMutator func(req *http.Request) error

// HTTPResponseValidator inspects the received response and can veto the probe.
type HTTPResponseValidator func(resp *http.Response) error

// HTTPProbeOption configures the behaviour of NewHTTPProbe.
type HTTPProbeOption func(*httpProbeConfig)

type httpProbeConfig struct {
	client             HTTPDoer
	expect             HTTPStatusExpectation
	requestMutators    []HTTPRequestMutator
	responseValidators []HTTPResponseValidator
	drainResponse      bool
}

func buildHTTPProbeConfig(client HTTPDoer, opts ...HTTPProbeOption) *httpProbeConfig {
	cfg := &httpProbeConfig{
		client:        client,
		expect:        defaultHTTPStatusExpectation,
		drainResponse: true,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}
	if cfg.client == nil {
		cfg.client = http.DefaultClient
	}
	if cfg.expect == nil {
		cfg.expect = defaultHTTPStatusExpectation
	}
	return cfg
}

func (c *httpProbeConfig) applyMutators(req *http.Request) error {
	for _, mutate := range c.requestMutators {
		if mutate == nil {
			continue
		}
		if err := mutate(req); err != nil {
			return err
		}
	}
	return nil
}

func (c *httpProbeConfig) validateResponse(resp *http.Response) error {
	if c.expect != nil && !c.expect(resp.StatusCode) {
		return fmt.Errorf("unexpected status %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}
	for _, validator := range c.responseValidators {
		if validator == nil {
			continue
		}
		if err := validator(resp); err != nil {
			return err
		}
	}
	return nil
}

// WithHTTPClient overrides the HTTP client used for the probe.
func WithHTTPClient(client HTTPDoer) HTTPProbeOption {
	return func(cfg *httpProbeConfig) {
		cfg.client = client
	}
}

// WithHTTPStatusExpectation installs a custom status validation function.
func WithHTTPStatusExpectation(expect HTTPStatusExpectation) HTTPProbeOption {
	return func(cfg *httpProbeConfig) {
		cfg.expect = expect
	}
}

// WithHTTPAllowedStatuses restricts the probe to succeed only for the provided status codes.
func WithHTTPAllowedStatuses(statuses ...int) HTTPProbeOption {
	allowed := make(map[int]struct{}, len(statuses))
	for _, status := range statuses {
		allowed[status] = struct{}{}
	}
	return func(cfg *httpProbeConfig) {
		cfg.expect = func(status int) bool {
			if len(allowed) == 0 {
				return defaultHTTPStatusExpectation(status)
			}
			_, ok := allowed[status]
			return ok
		}
	}
}

// WithHTTPRequestMutator registers a mutator that runs before the request is dispatched.
func WithHTTPRequestMutator(mutator HTTPRequestMutator) HTTPProbeOption {
	return func(cfg *httpProbeConfig) {
		cfg.requestMutators = append(cfg.requestMutators, mutator)
	}
}

// WithHTTPResponseValidator registers a validator that runs after a response is received.
func WithHTTPResponseValidator(validator HTTPResponseValidator) HTTPProbeOption {
	return func(cfg *httpProbeConfig) {
		cfg.responseValidators = append(cfg.responseValidators, validator)
	}
}

// WithHTTPDrainResponseBody toggles draining of the response body after validation.
func WithHTTPDrainResponseBody(enabled bool) HTTPProbeOption {
	return func(cfg *httpProbeConfig) {
		cfg.drainResponse = enabled
	}
}

package probe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Func represents a health check that returns an error when the resource is unavailable.
type Func func(ctx context.Context) error

// PingFunc represents a health check that returns an error when the resource is unavailable.
type PingFunc func(ctx context.Context) error

// DBPinger captures the subset of *sql.DB used for readiness checks.
type DBPinger interface {
	PingContext(ctx context.Context) error
}

// HTTPDoer represents the subset of *http.Client required by the HTTP probe helper.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewPingProbe wraps a PingFunc with standardised error handling suitable for InfoHandler probes.
func NewPingProbe(name string, fn PingFunc) Func {
	return func(ctx context.Context) error {
		if fn == nil {
			return fmt.Errorf("%s probe: ping function is nil", name)
		}
		if ctx == nil {
			ctx = context.Background()
		}

		if err := fn(ctx); err != nil {
			return fmt.Errorf("%s probe failed: %w", name, err)
		}
		return nil
	}
}

// MongoPinger captures the subset of the MongoDB client used for readiness checks.
type MongoPinger interface {
	Ping(ctx context.Context, rp *readpref.ReadPref) error
}

// NewMongoPingProbe creates a Func that pings MongoDB using the provided client.
// If readPref is nil it defaults to readpref.Primary.
func NewMongoPingProbe(client MongoPinger, readPref *readpref.ReadPref) Func {
	return func(ctx context.Context) error {
		if client == nil {
			return errors.New("mongo probe: client is nil")
		}

		if ctx == nil {
			ctx = context.Background()
		}

		rp := readPref
		if rp == nil {
			rp = readpref.Primary()
		}

		if err := client.Ping(ctx, rp); err != nil {
			return fmt.Errorf("mongo probe failed: %w", err)
		}
		return nil
	}
}

// NewDBPingProbe creates a Func that pings databases such as PostgreSQL using the provided client.
func NewDBPingProbe(name string, db DBPinger) Func {
	return func(ctx context.Context) error {
		if db == nil {
			return fmt.Errorf("%s probe: db client is nil", name)
		}
		if ctx == nil {
			ctx = context.Background()
		}

		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("%s probe failed: %w", name, err)
		}
		return nil
	}
}

// NewHTTPProbe creates a Func that performs an HTTP request against the supplied endpoint.
// The probe succeeds when the response status code is within the 2xx range.
func NewHTTPProbe(name, method, target string, client HTTPDoer) Func {
	return func(ctx context.Context) error {
		trimmedTarget := strings.TrimSpace(target)
		if trimmedTarget == "" {
			return fmt.Errorf("%s probe: target URL is required", name)
		}

		verb := strings.ToUpper(strings.TrimSpace(method))
		if verb == "" {
			verb = http.MethodGet
		}

		if ctx == nil {
			ctx = context.Background()
		}

		req, err := http.NewRequestWithContext(ctx, verb, trimmedTarget, nil)
		if err != nil {
			return fmt.Errorf("%s probe: failed to build request: %w", name, err)
		}

		d := client
		if d == nil {
			d = http.DefaultClient
		}

		resp, err := d.Do(req)
		if err != nil {
			return fmt.Errorf("%s probe request failed: %w", name, err)
		}
		defer resp.Body.Close()
		io.Copy(io.Discard, resp.Body)

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return fmt.Errorf("%s probe received status %d", name, resp.StatusCode)
		}
		return nil
	}
}

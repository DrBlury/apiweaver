package probe_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/drblury/apiweaver/probe"

	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type stubMongoPinger struct {
	err        error
	lastCtx    context.Context
	lastReadPF *readpref.ReadPref
}

func (s *stubMongoPinger) Ping(ctx context.Context, rp *readpref.ReadPref) error {
	s.lastCtx = ctx
	s.lastReadPF = rp
	return s.err
}

type stubDB struct {
	err error
}

func (s *stubDB) PingContext(ctx context.Context) error {
	return s.err
}

func TestNewPingProbe(t *testing.T) {
	t.Run("nil function", func(t *testing.T) {
		probeFunc := probe.NewPingProbe("db", nil)
		if err := probeFunc(context.Background()); err == nil {
			t.Fatal("expected error when ping function is nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		called := false
		probeFunc := probe.NewPingProbe("db", func(ctx context.Context) error {
			if ctx == nil {
				t.Fatal("expected non-nil context")
			}
			called = true
			return nil
		})

		if err := probeFunc(context.Background()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if !called {
			t.Fatal("expected ping function to be called")
		}
	})

	t.Run("failure", func(t *testing.T) {
		sentinel := errors.New("boom")
		probeFunc := probe.NewPingProbe("db", func(ctx context.Context) error {
			return sentinel
		})
		err := probeFunc(context.Background())
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected error to wrap sentinel, got %v", err)
		}
	})
}

func TestNewMongoPingProbe(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		probeFunc := probe.NewMongoPingProbe(nil, nil)
		if err := probeFunc(context.Background()); err == nil {
			t.Fatal("expected error when client is nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		stub := &stubMongoPinger{}
		probeFunc := probe.NewMongoPingProbe(stub, nil)
		if err := probeFunc(context.Background()); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if stub.lastCtx == nil {
			t.Fatal("expected context to be forwarded")
		}
		if stub.lastReadPF == nil {
			t.Fatal("expected read preference to be set")
		}
		if stub.lastReadPF.Mode() != readpref.PrimaryMode {
			t.Fatalf("expected primary read preference, got %v", stub.lastReadPF.Mode())
		}
	})

	t.Run("failure", func(t *testing.T) {
		sentinel := errors.New("unreachable")
		stub := &stubMongoPinger{err: sentinel}
		probeFunc := probe.NewMongoPingProbe(stub, readpref.Secondary())
		err := probeFunc(context.Background())
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel, got %v", err)
		}
		if stub.lastReadPF.Mode() != readpref.SecondaryMode {
			t.Fatalf("expected secondary read preference, got %v", stub.lastReadPF.Mode())
		}
	})
}

func ExampleNewPingProbe() {
	probeFunc := probe.NewPingProbe("noop", func(ctx context.Context) error {
		return nil
	})
	fmt.Println(probeFunc(context.Background()))
	// Output: <nil>
}

func ExampleNewDBPingProbe() {
	probeFunc := probe.NewDBPingProbe("postgres", &stubDB{})
	fmt.Println(probeFunc(context.Background()))
	// Output: <nil>
}

func ExampleNewHTTPProbe_defaultClient() {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	probeFunc := probe.NewHTTPProbe("docs", http.MethodGet, ts.URL, nil)
	fmt.Println(probeFunc(context.Background()))
	// Output: <nil>
}

func ExampleNewHTTPProbe() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	}))
	defer server.Close()

	probeFunc := probe.NewHTTPProbe("docs", http.MethodGet, server.URL, server.Client())
	fmt.Println(probeFunc(context.Background()))
	// Output: <nil>
}

func ExampleNewHTTPProbe_withOptions() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer demo" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("X-Version", "123")
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	probeFunc := probe.NewHTTPProbe(
		"docs",
		http.MethodGet,
		server.URL,
		nil,
		probe.WithHTTPRequestMutator(func(req *http.Request) error {
			req.Header.Set("Authorization", "Bearer demo")
			return nil
		}),
		probe.WithHTTPAllowedStatuses(http.StatusAccepted),
		probe.WithHTTPResponseValidator(func(resp *http.Response) error {
			if resp.Header.Get("X-Version") == "" {
				return errors.New("missing version header")
			}
			return nil
		}),
	)

	fmt.Println(probeFunc(context.Background()))
	// Output: <nil>
}

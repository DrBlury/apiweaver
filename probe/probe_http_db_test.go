package probe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type stubDBPinger struct {
	err     error
	lastCtx context.Context
}

func (s *stubDBPinger) PingContext(ctx context.Context) error {
	s.lastCtx = ctx
	return s.err
}

type stubHTTPClient struct {
	resp    *http.Response
	err     error
	lastReq *http.Request
}

func (s *stubHTTPClient) Do(req *http.Request) (*http.Response, error) {
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return s.resp, nil
}

func TestNewDBPingProbe(t *testing.T) {
	t.Run("nil client", func(t *testing.T) {
		probeFunc := NewDBPingProbe("postgres", nil)
		if err := probeFunc(context.Background()); err == nil {
			t.Fatal("expected error when db client is nil")
		}
	})

	t.Run("success", func(t *testing.T) {
		stub := &stubDBPinger{}
		probeFunc := NewDBPingProbe("postgres", stub)
		if err := probeFunc(nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if stub.lastCtx == nil {
			t.Fatal("expected context to be supplied")
		}
	})

	t.Run("failure wraps error", func(t *testing.T) {
		sentinel := errors.New("unreachable")
		stub := &stubDBPinger{err: sentinel}
		probeFunc := NewDBPingProbe("postgres", stub)
		err := probeFunc(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel, got %v", err)
		}
	})
}

func TestNewHTTPProbe(t *testing.T) {
	t.Run("requires target", func(t *testing.T) {
		probeFunc := NewHTTPProbe("search", http.MethodGet, "", nil)
		if err := probeFunc(context.Background()); err == nil {
			t.Fatal("expected error when target missing")
		}
	})

	t.Run("success with default client", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("expected GET request, got %s", r.Method)
			}
			io.WriteString(w, "ok")
		}))
		defer server.Close()

		probeFunc := NewHTTPProbe("docs", "", server.URL, nil)
		if err := probeFunc(nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("non success status fails", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("oops")),
		}
		client := &stubHTTPClient{resp: resp}
		probeFunc := NewHTTPProbe("docs", http.MethodHead, "https://example.invalid", client)

		err := probeFunc(context.Background())
		if err == nil {
			t.Fatal("expected error when status not 2xx")
		}
		if client.lastReq == nil || client.lastReq.Method != http.MethodHead {
			t.Fatalf("expected HEAD request, got %+v", client.lastReq)
		}
	})

	t.Run("request failure is propagated", func(t *testing.T) {
		sentinel := errors.New("network down")
		client := &stubHTTPClient{err: sentinel}
		probeFunc := NewHTTPProbe("docs", http.MethodGet, "https://example.invalid", client)

		err := probeFunc(context.Background())
		if err == nil {
			t.Fatal("expected error")
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel, got %v", err)
		}
	})
}

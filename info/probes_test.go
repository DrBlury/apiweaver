package info

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestInfoHandler_respondProbe(t *testing.T) {
	handler := NewInfoHandler()
	req := httptest.NewRequest(http.MethodGet, "/probe", nil)
	rr := httptest.NewRecorder()

	handler.respondProbe(rr, req, http.StatusAccepted, "WARN", "db", "search")

	if rr.Code != http.StatusAccepted {
		t.Fatalf("expected status %d, got %d", http.StatusAccepted, rr.Code)
	}

	payload := decodeProbePayload(t, rr.Body.Bytes())
	if payload.Status != "WARN" {
		t.Fatalf("expected status WARN, got %s", payload.Status)
	}
	expectedDetails := []string{"db", "search"}
	if !reflect.DeepEqual(payload.Details, expectedDetails) {
		t.Fatalf("expected details %v, got %v", expectedDetails, payload.Details)
	}
}

func TestInfoHandler_runChecks(t *testing.T) {
	handler := NewInfoHandler()

	t.Run("no checks", func(t *testing.T) {
		if err := handler.runChecks(context.Background(), nil); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("skips nil checks", func(t *testing.T) {
		checks := []ProbeFunc{nil, func(context.Context) error { return nil }}
		if err := handler.runChecks(context.Background(), checks); err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("returns wrapped errors", func(t *testing.T) {
		sentinel := errors.New("boom")
		err := handler.runChecks(context.Background(), []ProbeFunc{func(context.Context) error { return sentinel }})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "probe 1 failed") {
			t.Fatalf("expected error message to describe probe failure, got %v", err)
		}
		if !errors.Is(err, sentinel) {
			t.Fatalf("expected wrapped sentinel, got %v", err)
		}
	})

	t.Run("propagates deadline exceeded", func(t *testing.T) {
		err := handler.runChecks(context.Background(), []ProbeFunc{func(context.Context) error {
			return context.DeadlineExceeded
		}})
		if err == nil || !strings.Contains(err.Error(), "timed out") {
			t.Fatalf("expected timeout error, got %v", err)
		}
	})

	t.Run("propagates cancellation", func(t *testing.T) {
		err := handler.runChecks(context.Background(), []ProbeFunc{func(context.Context) error {
			return context.Canceled
		}})
		if err == nil || !strings.Contains(err.Error(), "was cancelled") {
			t.Fatalf("expected cancellation error, got %v", err)
		}
	})

	t.Run("all probes must succeed", func(t *testing.T) {
		called := 0
		err := handler.runChecks(context.Background(), []ProbeFunc{
			func(context.Context) error {
				called++
				return nil
			},
			func(context.Context) error {
				called++
				return nil
			},
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if called != 2 {
			t.Fatalf("expected both probes to run, ran %d", called)
		}
	})
}

func TestFilterProbes(t *testing.T) {
	fn1 := func(context.Context) error { return nil }
	fn2 := func(context.Context) error { return nil }

	t.Run("returns nil when no probes provided", func(t *testing.T) {
		if filtered := filterProbes(nil); filtered != nil {
			t.Fatalf("expected nil slice, got %v", filtered)
		}
	})

	t.Run("strips nil entries", func(t *testing.T) {
		filtered := filterProbes([]ProbeFunc{nil, fn1, nil, fn2})
		if filtered == nil {
			t.Fatal("expected filtered slice")
		}
		if len(filtered) != 2 {
			t.Fatalf("expected two probes, got %d", len(filtered))
		}
		if reflect.ValueOf(filtered[0]).Pointer() != reflect.ValueOf(fn1).Pointer() {
			t.Fatalf("expected first probe to be fn1")
		}
		if reflect.ValueOf(filtered[1]).Pointer() != reflect.ValueOf(fn2).Pointer() {
			t.Fatalf("expected second probe to be fn2")
		}
	})

	t.Run("returns nil when all entries are nil", func(t *testing.T) {
		if filtered := filterProbes([]ProbeFunc{nil, nil}); filtered != nil {
			t.Fatalf("expected nil slice, got %v", filtered)
		}
	})
}

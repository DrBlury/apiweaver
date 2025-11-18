package router_test

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/drblury/apiweaver/router"
)

func ExampleNew_customOptions() {
	apiHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "hello")
	})

	mux := router.New(
		apiHandler,
		router.WithLogger(slog.New(slog.NewJSONHandler(io.Discard, nil))),
		router.WithConfig(router.Config{
			Timeout: 2 * time.Second,
			CORS: router.CORSConfig{
				Origins: []string{"https://example.com"},
				Methods: []string{http.MethodGet, http.MethodOptions},
				Headers: []string{"Content-Type"},
			},
			HideHeaders: []string{"Authorization"},
		}),
		router.WithMiddlewares(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Stage", "prepend")
				next.ServeHTTP(w, r)
			})
		}),
		router.WithTrailingMiddlewares(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
				w.Header().Set("X-Chain", "completed")
			})
		}),
	)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	fmt.Println(rec.Header().Get("Access-Control-Allow-Origin"))
	fmt.Println(rec.Header().Get("X-Stage"))
	fmt.Println(rec.Header().Get("X-Chain"))
	fmt.Println(strings.TrimSpace(rec.Body.String()))

	// Output:
	// https://example.com
	// prepend
	// completed
	// hello
}

func ExampleWithMiddlewareChain() {
	records := make([]string, 0, 4)
	middleware := func(label string) router.Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				records = append(records, label+"-before")
				next.ServeHTTP(w, r)
				records = append(records, label+"-after")
			})
		}
	}

	mux := router.New(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "ok")
		}),
		router.WithMiddlewareChain(
			middleware("first"),
			middleware("second"),
		),
	)

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	fmt.Println(rec.Code)
	fmt.Println(records)

	// Output:
	// 200
	// [first-before second-before second-after first-after]
}

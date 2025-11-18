package responder_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/drblury/apiweaver/responder"
)

func ExampleResponder_full() {
	errDuplicate := errors.New("project already exists")
	r := responder.NewResponder(
		responder.WithErrorClassifier(func(err error) (int, bool) {
			if errors.Is(err, errDuplicate) {
				return http.StatusConflict, true
			}
			return 0, false
		}),
	)

	store := make(map[string]struct{})
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Name string `json:"name"`
		}
		if !r.ReadRequestBody(w, req, &body) {
			return
		}
		if body.Name == "" {
			r.HandleBadRequestError(w, req, errors.New("name is required"))
			return
		}
		if _, exists := store[body.Name]; exists {
			r.HandleErrors(w, req, errDuplicate)
			return
		}
		store[body.Name] = struct{}{}
		r.RespondWithJSON(w, req, http.StatusCreated, map[string]string{"name": body.Name})
	})

	createReq := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"weaver"}`))
	createRec := httptest.NewRecorder()
	handler.ServeHTTP(createRec, createReq)
	fmt.Println(createRec.Code)
	fmt.Println(strings.TrimSpace(createRec.Body.String()))

	dupReq := httptest.NewRequest(http.MethodPost, "/projects", strings.NewReader(`{"name":"weaver"}`))
	dupRec := httptest.NewRecorder()
	handler.ServeHTTP(dupRec, dupReq)

	var problem responder.ProblemDetails
	_ = json.Unmarshal(dupRec.Body.Bytes(), &problem)
	fmt.Println(problem.Status)
	fmt.Println(problem.Title)

	// Output:
	// 201
	// {"name":"weaver"}
	// 409
	// Conflict
}

func ExampleWithStatusMetadata() {
	r := responder.NewResponder(
		responder.WithStatusMetadata(http.StatusUnauthorized, responder.StatusMetadata{
			Title:   "Missing token",
			LogMsg:  "authentication failed",
			TypeURI: "https://status.example.com/unauthorized",
		}),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/secret", nil)
	r.HandleUnauthorizedError(rec, req, errors.New("token expired"))

	fmt.Println(rec.Code)
	fmt.Println(strings.Contains(rec.Body.String(), "\"title\":\"Missing token\""))

	// Output:
	// 401
	// true
}

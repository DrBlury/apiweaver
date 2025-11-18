package responder

import (
	"net/http"

	"github.com/drblury/apiweaver/jsonutil"
)

// HandleAPIError renders a structured JSON response for the supplied HTTP
// status and logs the payload using the configured logger.
func (r *Responder) HandleAPIError(w http.ResponseWriter, req *http.Request, status int, err error, logMsg ...string) {
	if err == nil {
		return
	}

	meta := r.statusMetaFor(status)
	problem := r.buildProblemDetails(req, status, err, meta)
	r.logProblem(req, meta, err, problem.TraceID, status, logMsg)
	r.respondWithJSON(w, req, status, problem, problemContentType)
}

// HandleInternalServerError is a shortcut that reports a 500 status code.
func (r *Responder) HandleInternalServerError(w http.ResponseWriter, req *http.Request, err error, logMsg ...string) {
	r.HandleAPIError(w, req, http.StatusInternalServerError, err, logMsg...)
}

// HandleBadRequestError reports client validation errors using HTTP 400.
func (r *Responder) HandleBadRequestError(w http.ResponseWriter, req *http.Request, err error, logMsg ...string) {
	r.HandleAPIError(w, req, http.StatusBadRequest, err, logMsg...)
}

// HandleUnauthorizedError reports authentication failures using HTTP 401.
func (r *Responder) HandleUnauthorizedError(w http.ResponseWriter, req *http.Request, err error, logMsg ...string) {
	r.HandleAPIError(w, req, http.StatusUnauthorized, err, logMsg...)
}

// RespondWithJSON serialises the provided value and writes it to the response
// using the supplied status code.
func (r *Responder) RespondWithJSON(w http.ResponseWriter, req *http.Request, status int, v any) {
	r.respondWithJSON(w, req, status, v, jsonContentType)
}

// HandleErrors inspects the supplied error using the configured classifier and
// emits an appropriate JSON response.
func (r *Responder) HandleErrors(w http.ResponseWriter, req *http.Request, err error, msgs ...string) {
	if err == nil {
		return
	}

	if status, handled := r.classifyError(err); handled {
		r.HandleAPIError(w, req, status, err, msgs...)
		return
	}

	r.HandleInternalServerError(w, req, err, msgs...)
}

func (r *Responder) respondWithJSON(w http.ResponseWriter, req *http.Request, status int, payload any, contentType string) {
	if w == nil {
		return
	}

	body, err := r.marshalPayload(payload)
	if err != nil {
		r.logger().Error("failed to encode response", "error", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	r.writeResponse(w, status, resolveContentType(contentType, jsonContentType), body)
}

func (r *Responder) marshalPayload(payload any) ([]byte, error) {
	data, err := jsonutil.Marshal(payload)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 || data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}
	return data, nil
}

func (r *Responder) writeResponse(w http.ResponseWriter, status int, contentType string, body []byte) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	if _, err := w.Write(body); err != nil {
		r.logger().Error("failed to write response", "error", err)
	}
}

func resolveContentType(provided, fallback string) string {
	if provided == "" {
		return fallback
	}
	return provided
}

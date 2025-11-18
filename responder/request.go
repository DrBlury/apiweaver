package responder

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/drblury/apiweaver/jsonutil"
)

// ReadRequestBody parses the request body into the provided value and handles
// malformed content by returning a JSON error response.
func (r *Responder) ReadRequestBody(w http.ResponseWriter, req *http.Request, v any) bool {
	if err := r.decodeRequestBody(req, v); err != nil {
		r.HandleBadRequestError(w, req, err, "failed to parse request body")
		return false
	}
	return true
}

func (r *Responder) decodeRequestBody(req *http.Request, v any) error {
	if req == nil || req.Body == nil {
		return errors.New("request body is required")
	}
	if err := jsonutil.Decode(req.Body, v); err != nil {
		if errors.Is(err, io.EOF) {
			return io.ErrUnexpectedEOF
		}
		return err
	}
	return nil
}

func requestInstance(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}
	return req.URL.RequestURI()
}

func requestContext(req *http.Request) context.Context {
	if req == nil {
		return context.Background()
	}
	return req.Context()
}

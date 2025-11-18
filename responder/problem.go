package responder

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ProblemDetails aligns HTTP error responses with RFC 9457 problem documents.
type ProblemDetails struct {
	Type      string `json:"type,omitempty"`
	Title     string `json:"title"`
	Status    int    `json:"status"`
	Detail    string `json:"detail,omitempty"`
	Instance  string `json:"instance,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

func (r *Responder) statusMetaFor(status int) statusMeta {
	meta, ok := r.statusMetadata[status]
	if !ok {
		meta = statusMeta{}
	}
	return normalizeStatusMeta(status, meta)
}

func (r *Responder) buildProblemDetails(req *http.Request, status int, err error, meta statusMeta) ProblemDetails {
	return ProblemDetails{
		Type:      meta.typeURI,
		Title:     meta.title,
		Status:    status,
		Detail:    err.Error(),
		Instance:  requestInstance(req),
		TraceID:   newTraceID(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
}

func (r *Responder) logProblem(req *http.Request, meta statusMeta, err error, traceID string, status int, msgs []string) {
	logger := r.logger().With("error", err.Error(), "traceId", traceID, "status", status)
	if len(msgs) > 0 {
		logger = logger.With("logMessages", msgs)
	}
	logger.Log(requestContext(req), meta.logLevel, meta.logMsg)
}

func normalizeStatusMeta(status int, meta statusMeta) statusMeta {
	if meta.logLevel == 0 {
		meta.logLevel = slog.LevelError
	}
	if meta.title == "" {
		meta.title = http.StatusText(status)
	}
	if meta.logMsg == "" {
		meta.logMsg = meta.title
	}
	if meta.typeURI == "" {
		meta.typeURI = fmt.Sprintf("%s/%d", statusDocBaseURL, status)
	}
	return meta
}

package responder

import (
	"log/slog"
	"net/http"
)

const (
	jsonContentType    = "application/json"
	problemContentType = "application/problem+json"
	statusDocBaseURL   = "https://httpstatuses.io"
)

// ErrorClassifierFunc inspects an error and returns the HTTP status that should
// be used for the response. The boolean indicates whether the error was
// classified and prevents the generic internal server handler from running.
type ErrorClassifierFunc func(err error) (status int, handled bool)

// ResponderOption follows the functional options pattern used by NewResponder
// to configure optional collaborators.
type ResponderOption func(*Responder)

type statusMeta struct {
	typeURI  string
	title    string
	logLevel slog.Level
	logMsg   string
}

// StatusMetadata allows callers to customise how particular HTTP status codes
// are logged and represented in error payloads.
type StatusMetadata struct {
	TypeURI  string
	Title    string
	LogLevel slog.Level
	LogMsg   string
}

// Responder centralises error handling, JSON rendering, and logging for HTTP
// handlers. It provides structured error payloads with correlation identifiers
// and consistent log records.
type Responder struct {
	log             *slog.Logger
	statusMetadata  map[int]statusMeta
	errorClassifier ErrorClassifierFunc
}

// NewResponder constructs a Responder with default status metadata and the
// global slog logger. Use ResponderOption functions to override specific
// behaviours.
func NewResponder(opts ...ResponderOption) *Responder {
	r := &Responder{
		log:            slog.Default(),
		statusMetadata: defaultStatusMetadata(),
	}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

// WithLogger injects a custom slog logger for error reporting and payload
// logging.
func WithLogger(logger *slog.Logger) ResponderOption {
	return func(r *Responder) {
		if logger != nil {
			r.log = logger
		}
	}
}

// WithErrorClassifier installs a classifier used by HandleErrors to derive the
// HTTP status code from returned errors.
func WithErrorClassifier(classifier ErrorClassifierFunc) ResponderOption {
	return func(r *Responder) {
		r.errorClassifier = classifier
	}
}

// WithStatusMetadata overrides the error metadata used for a specific HTTP
// status code.
func WithStatusMetadata(status int, meta StatusMetadata) ResponderOption {
	return func(r *Responder) {
		if r.statusMetadata == nil {
			r.statusMetadata = make(map[int]statusMeta)
		}
		level := meta.LogLevel
		if level == 0 {
			level = slog.LevelError
		}
		title := meta.Title
		if title == "" {
			title = http.StatusText(status)
		}
		msg := meta.LogMsg
		if msg == "" {
			msg = title
		}
		r.statusMetadata[status] = statusMeta{
			typeURI:  meta.TypeURI,
			title:    title,
			logLevel: level,
			logMsg:   msg,
		}
	}
}

// Logger returns the slog logger used internally by the responder.
func (r *Responder) Logger() *slog.Logger {
	return r.logger()
}

func (r *Responder) logger() *slog.Logger {
	if r == nil || r.log == nil {
		return slog.Default()
	}
	return r.log
}

func (r *Responder) classifyError(err error) (int, bool) {
	if r.errorClassifier == nil {
		return 0, false
	}
	return r.errorClassifier(err)
}

func defaultStatusMetadata() map[int]statusMeta {
	return map[int]statusMeta{
		http.StatusInternalServerError: {title: http.StatusText(http.StatusInternalServerError), logLevel: slog.LevelError, logMsg: "Internal Server Error"},
		http.StatusBadRequest:          {title: http.StatusText(http.StatusBadRequest), logLevel: slog.LevelWarn, logMsg: "Bad Request"},
		http.StatusUnauthorized:        {title: http.StatusText(http.StatusUnauthorized), logLevel: slog.LevelWarn, logMsg: "Unauthorized"},
	}
}

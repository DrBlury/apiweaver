package info

import (
	"errors"
	"net/http"
)

// GetStatus returns a simple health payload that can be used for lightweight diagnostics.
func (ih *InfoHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ih.respondProbe(w, r, http.StatusOK, "HEALTHY")
}

// GetHealthz implements the liveness probe recommended for Kubernetes.
func (ih *InfoHandler) GetHealthz(w http.ResponseWriter, r *http.Request) {
	if err := ih.runChecks(r.Context(), ih.livenessChecks); err != nil {
		ih.HandleAPIError(w, r, http.StatusServiceUnavailable, err, "liveness probe failed")
		return
	}
	ih.respondProbe(w, r, http.StatusOK, "ok")
}

// GetReadyz implements the readiness probe recommended for Kubernetes.
func (ih *InfoHandler) GetReadyz(w http.ResponseWriter, r *http.Request) {
	if err := ih.runChecks(r.Context(), ih.readinessChecks); err != nil {
		ih.HandleAPIError(w, r, http.StatusServiceUnavailable, err, "readiness probe failed")
		return
	}
	ih.respondProbe(w, r, http.StatusOK, "ready")
}

// GetVersion returns the structure provided by the configured InfoProvider.
func (ih *InfoHandler) GetVersion(w http.ResponseWriter, r *http.Request) {
	payload := ih.infoProvider()
	if payload == nil {
		payload = map[string]string{}
	}
	ih.RespondWithJSON(w, r, http.StatusOK, payload)
}

// GetOpenAPIJSON streams the configured OpenAPI JSON document to the caller.
func (ih *InfoHandler) GetOpenAPIJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bytes, err := ih.swaggerProvider()
	if err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to load swagger spec")
		return
	}

	if _, err = w.Write(bytes); err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to write swagger response")
		return
	}
}

// GetOpenAPIHTML renders an embedded Stoplight viewer that fetches the OpenAPI document from the JSON endpoint.
func (ih *InfoHandler) GetOpenAPIHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if ih.openapiTemplate == nil {
		err := errors.New("openapi template not configured")
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to render openapi template")
		return
	}

	var data any
	if ih.dataProvider != nil {
		data = ih.dataProvider(r, ih.baseURL)
	}
	if data == nil {
		data = defaultTemplateDataProvider(r, ih.baseURL)
	}

	if err := ih.openapiTemplate.Execute(w, data); err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to render openapi template")
		return
	}
}

// GetAsyncAPIJSON streams the configured AsyncAPI JSON document to the caller.
func (ih *InfoHandler) GetAsyncAPIJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	bytes, err := ih.asyncapiProvider()
	if err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to load asyncapi spec")
		return
	}

	if _, err = w.Write(bytes); err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to write asyncapi response")
		return
	}
}

// GetAsyncAPIHTML renders an embedded AsyncAPI React Component viewer that fetches the AsyncAPI document from the JSON endpoint.
func (ih *InfoHandler) GetAsyncAPIHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	if ih.asyncapiTemplate == nil {
		err := errors.New("asyncapi template not configured")
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to render asyncapi template")
		return
	}

	var data any
	if ih.asyncapiDataProvider != nil {
		data = ih.asyncapiDataProvider(r, ih.baseURL)
	}
	if data == nil {
		data = defaultAsyncAPITemplateDataProvider(r, ih.baseURL)
	}

	if err := ih.asyncapiTemplate.Execute(w, data); err != nil {
		ih.HandleAPIError(w, r, http.StatusInternalServerError, err, "failed to render asyncapi template")
		return
	}
}

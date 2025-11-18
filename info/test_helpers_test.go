package info

import (
	"encoding/json"
	"testing"

	"github.com/drblury/apiweaver/responder"
)

func decodeProbePayload(t *testing.T, body []byte) probePayload {
	t.Helper()

	var payload probePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to decode probe payload: %v (body: %s)", err, string(body))
	}
	return payload
}

func decodeProblemDetails(t *testing.T, body []byte) responder.ProblemDetails {
	t.Helper()

	var problem responder.ProblemDetails
	if err := json.Unmarshal(body, &problem); err != nil {
		t.Fatalf("failed to decode problem details: %v (body: %s)", err, string(body))
	}
	return problem
}

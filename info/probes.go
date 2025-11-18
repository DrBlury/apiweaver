package info

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

type probePayload struct {
	Status  string   `json:"status"`
	Details []string `json:"details,omitempty"`
}

func (ih *InfoHandler) respondProbe(w http.ResponseWriter, r *http.Request, statusCode int, state string, details ...string) {
	payload := probePayload{Status: state}
	if len(details) > 0 {
		payload.Details = append(payload.Details, details...)
	}
	ih.RespondWithJSON(w, r, statusCode, payload)
}

func (ih *InfoHandler) runChecks(ctx context.Context, checks []ProbeFunc) error {
	if len(checks) == 0 {
		return nil
	}

	timeout := ih.probeTimeout
	if timeout <= 0 {
		timeout = defaultProbeTimeout
	}

	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for idx, check := range checks {
		if check == nil {
			continue
		}

		if err := check(probeCtx); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				return fmt.Errorf("probe %d timed out after %s", idx+1, timeout)
			}
			if errors.Is(err, context.Canceled) {
				return fmt.Errorf("probe %d was cancelled", idx+1)
			}
			return fmt.Errorf("probe %d failed: %w", idx+1, err)
		}
	}

	return nil
}

func filterProbes(checks []ProbeFunc) []ProbeFunc {
	if len(checks) == 0 {
		return nil
	}

	filtered := make([]ProbeFunc, 0, len(checks))
	for _, check := range checks {
		if check != nil {
			filtered = append(filtered, check)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	return filtered
}

package probe

import (
	"context"
	"fmt"
)

func contextOrBackground(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func defaultHTTPStatusExpectation(status int) bool {
	return status >= 200 && status < 300
}

func nilComponentError(name, component string) error {
	return fmt.Errorf("%s probe: %s is nil", name, component)
}

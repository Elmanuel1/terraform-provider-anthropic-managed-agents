package auth

import (
	"context"
	"net/http"
)

// Credentials authenticates an outbound HTTP request.
type Credentials interface {
	Authenticate(ctx context.Context, req *http.Request) error
}

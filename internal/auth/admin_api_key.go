package auth

import (
	"context"
	"fmt"
	"net/http"
)

// AdminAPIKey authenticates using a static Anthropic Admin API key.
// Wrap with WithBeta to set the anthropic-beta header for a specific endpoint.
type AdminAPIKey struct {
	Key string
}

func (a AdminAPIKey) Authenticate(_ context.Context, req *http.Request) error {
	if a.Key == "" {
		return fmt.Errorf("admin API key is empty")
	}
	req.Header.Set(HeaderAPIKey, a.Key)
	req.Header.Set(HeaderVersion, APIVersion)
	return nil
}

package auth

import (
	"context"
	"fmt"
	"net/http"
)

// AdminAPIKey authenticates using a static Anthropic Admin API key.
// Beta defaults to AdminBeta; set it to override (e.g. AgentsBeta for memory stores).
type AdminAPIKey struct {
	Key  string
	Beta string
}

func (a AdminAPIKey) Authenticate(_ context.Context, req *http.Request) error {
	if a.Key == "" {
		return fmt.Errorf("admin API key is empty")
	}
	beta := a.Beta
	if beta == "" {
		beta = AdminBeta
	}
	req.Header.Set(HeaderAPIKey, a.Key)
	req.Header.Set(HeaderVersion, APIVersion)
	req.Header.Set(HeaderBeta, beta)
	return nil
}

package auth

import (
	"context"
	"fmt"
	"net/http"
)

// WorkspaceAPIKey authenticates using a workspace-scoped Anthropic API key.
type WorkspaceAPIKey struct {
	Key string
}

func (w WorkspaceAPIKey) Authenticate(_ context.Context, req *http.Request) error {
	if w.Key == "" {
		return fmt.Errorf("workspace API key is empty")
	}
	req.Header.Set(HeaderAPIKey, w.Key)
	req.Header.Set(HeaderVersion, APIVersion)
	req.Header.Set(HeaderBeta, AgentsBeta)
	return nil
}

package auth

import (
	"context"
	"fmt"
	"net/http"
)

// WorkspaceAPIKey authenticates using a workspace-scoped Anthropic API key.
// Beta defaults to AgentsBeta; set it to override (e.g. SkillsBeta for skills endpoints).
type WorkspaceAPIKey struct {
	Key  string
	Beta string
}

func (w WorkspaceAPIKey) Authenticate(_ context.Context, req *http.Request) error {
	if w.Key == "" {
		return fmt.Errorf("workspace API key is empty")
	}
	beta := w.Beta
	if beta == "" {
		beta = AgentsBeta
	}
	req.Header.Set(HeaderAPIKey, w.Key)
	req.Header.Set(HeaderVersion, APIVersion)
	req.Header.Set(HeaderBeta, beta)
	return nil
}

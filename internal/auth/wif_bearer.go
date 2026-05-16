package auth

import (
	"context"
	"fmt"
	"net/http"
)

// WIFBearer authenticates using a minted WIF token scoped to a workspace.
type WIFBearer struct {
	Config      *WIFConfig
	WorkspaceID string
}

func (w WIFBearer) Authenticate(ctx context.Context, req *http.Request) error {
	if w.Config == nil {
		return fmt.Errorf("missing WIF config")
	}
	tok, err := MintToken(ctx, w.Config, w.WorkspaceID)
	if err != nil {
		return fmt.Errorf("minting WIF token: %w", err)
	}
	req.Header.Set(HeaderAuth, "Bearer "+tok.AccessToken)
	req.Header.Set(HeaderVersion, APIVersion)
	req.Header.Set(HeaderBeta, AgentsBeta)
	return nil
}

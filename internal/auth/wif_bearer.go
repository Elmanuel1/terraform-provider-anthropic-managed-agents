package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
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
	if strings.TrimSpace(w.WorkspaceID) == "" {
		return fmt.Errorf("workspace ID is empty")
	}

	// Return cached token if still valid (with 30-second early-expiry buffer).
	// The OIDC JWT has a one-time-use JTI, so all parallel resource creates must
	// share the single minted access token rather than each exchanging the JWT.
	if cached, ok := w.Config.tokenCache.Load(w.WorkspaceID); ok {
		tok := cached.(*MintedToken)
		if time.Now().Before(tok.ExpiresAt.Add(-30 * time.Second)) {
			req.Header.Set(HeaderAuth, "Bearer "+tok.AccessToken)
			req.Header.Set(HeaderVersion, APIVersion)
			return nil
		}
	}

	tok, err := MintToken(ctx, w.Config, w.WorkspaceID)
	if err != nil {
		return fmt.Errorf("minting WIF token: %w", err)
	}
	w.Config.tokenCache.Store(w.WorkspaceID, tok)
	req.Header.Set(HeaderAuth, "Bearer "+tok.AccessToken)
	req.Header.Set(HeaderVersion, APIVersion)
	return nil
}

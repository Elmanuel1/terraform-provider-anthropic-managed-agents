package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var anthropicTokenURL = "https://api.anthropic.com/v1/oauth/token"

type WIFConfig struct {
	FederationRuleID string
	OrganizationID   string
	ServiceAccountID string
	jwt              string   // TFC-injected OIDC token, valid for the run; one-time-use JTI
	tokenCache       sync.Map // key: workspaceID → *MintedToken; prevents JTI reuse across parallel creates
}

type MintedToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

// NewWIFConfig builds a WIFConfig from already-resolved values.
// The JWT is always read from the environment (TFC injects it automatically).
func NewWIFConfig(ruleID, orgID, svcID string) (*WIFConfig, error) {
	jwt := os.Getenv("TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC")
	if jwt == "" {
		jwt = os.Getenv("TFC_WORKLOAD_IDENTITY_TOKEN")
	}

	if ruleID == "" && orgID == "" && svcID == "" && jwt == "" {
		return nil, nil
	}

	var missing []string
	if ruleID == "" {
		missing = append(missing, "federation_rule_id")
	}
	if orgID == "" {
		missing = append(missing, "organization_id")
	}
	if svcID == "" {
		missing = append(missing, "service_account_id")
	}
	if jwt == "" {
		missing = append(missing, "TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC (or TFC_WORKLOAD_IDENTITY_TOKEN)")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("incomplete WIF configuration, missing: %s", strings.Join(missing, ", "))
	}

	return &WIFConfig{
		FederationRuleID: ruleID,
		OrganizationID:   orgID,
		ServiceAccountID: svcID,
		jwt:              jwt,
	}, nil
}


func jwtClaims(token string) (sub, aud string) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "<invalid-jwt>", "<invalid-jwt>"
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "<decode-error>", "<decode-error>"
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "<parse-error>", "<parse-error>"
	}
	sub, _ = claims["sub"].(string)
	switch v := claims["aud"].(type) {
	case string:
		aud = v
	case []any:
		var parts []string
		for _, a := range v {
			if s, ok := a.(string); ok {
				parts = append(parts, s)
			}
		}
		aud = strings.Join(parts, ",")
	}
	return sub, aud
}

func LogJWTClaims(ctx context.Context, cfg *WIFConfig) {
	if cfg == nil {
		return
	}
	parts := strings.Split(cfg.jwt, ".")
	if len(parts) < 2 {
		tflog.Warn(ctx, "TFC OIDC token does not look like a JWT")
		return
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		tflog.Warn(ctx, "failed to decode JWT payload", map[string]any{"error": err.Error()})
		return
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		tflog.Warn(ctx, "failed to parse JWT claims", map[string]any{"error": err.Error()})
		return
	}
	tflog.Info(ctx, "TFC OIDC token claims", map[string]any{
		"sub": claims["sub"],
		"aud": claims["aud"],
		"iss": claims["iss"],
	})
	tflog.Info(ctx, "WIF config", map[string]any{
		"federation_rule_id": cfg.FederationRuleID,
		"organization_id":    cfg.OrganizationID,
		"service_account_id": cfg.ServiceAccountID,
	})
}

func MintToken(ctx context.Context, cfg *WIFConfig, workspaceID string) (*MintedToken, error) {
	if cfg == nil {
		return nil, fmt.Errorf("missing WIF config")
	}
	body, err := json.Marshal(map[string]string{
		"grant_type":         "urn:ietf:params:oauth:grant-type:jwt-bearer",
		"assertion":          cfg.jwt,
		"federation_rule_id": cfg.FederationRuleID,
		"organization_id":    cfg.OrganizationID,
		"service_account_id": cfg.ServiceAccountID,
		"workspace_id":       workspaceID,
	})
	if err != nil {
		return nil, fmt.Errorf("building exchange request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicTokenURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building HTTP request: %w", err)
	}
	req.Header.Set(HeaderContentType, MIMEApplicationJSON)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		sub, aud := jwtClaims(cfg.jwt)
		return nil, fmt.Errorf("token exchange returned HTTP %d: %s\n  jwt.sub=%s jwt.aud=%s\n  federation_rule_id=%s organization_id=%s service_account_id=%s",
			resp.StatusCode, raw, sub, aud, cfg.FederationRuleID, cfg.OrganizationID, cfg.ServiceAccountID)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	if result.AccessToken == "" {
		return nil, fmt.Errorf("token exchange returned empty access_token")
	}

	return &MintedToken{
		AccessToken: result.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

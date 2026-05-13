package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const anthropicTokenURL = "https://api.anthropic.com/v1/oauth/token"

type wifConfig struct {
	FederationRuleID string
	OrganizationID   string
	ServiceAccountID string
	jwt              string // TFC-injected OIDC token, valid for the run
}

type mintedToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

func readWIFConfig() (*wifConfig, error) {
	rule := os.Getenv("ANTHROPIC_FEDERATION_RULE_ID")
	org := os.Getenv("ANTHROPIC_ORGANIZATION_ID")
	svc := os.Getenv("ANTHROPIC_SERVICE_ACCOUNT_ID")
	jwt := os.Getenv("TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC")

	// Not configured — no WIF
	if rule == "" && org == "" && svc == "" && jwt == "" {
		return nil, nil
	}

	var missing []string
	if rule == "" {
		missing = append(missing, "ANTHROPIC_FEDERATION_RULE_ID")
	}
	if org == "" {
		missing = append(missing, "ANTHROPIC_ORGANIZATION_ID")
	}
	if svc == "" {
		missing = append(missing, "ANTHROPIC_SERVICE_ACCOUNT_ID")
	}
	if jwt == "" {
		missing = append(missing, "TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("incomplete WIF configuration, missing: %v", missing)
	}

	return &wifConfig{
		FederationRuleID: rule,
		OrganizationID:   org,
		ServiceAccountID: svc,
		jwt:              jwt,
	}, nil
}

func mintToken(ctx context.Context, cfg *wifConfig, workspaceID string) (*mintedToken, error) {
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
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token exchange request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange returned HTTP %d: %s", resp.StatusCode, raw)
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

	return &mintedToken{
		AccessToken: result.AccessToken,
		ExpiresAt:   time.Now().Add(time.Duration(result.ExpiresIn) * time.Second),
	}, nil
}

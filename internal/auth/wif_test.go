package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestReadWIFConfig_NotConfigured(t *testing.T) {
	clearWIFEnv(t)
	cfg, err := ReadWIFConfig()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %+v", cfg)
	}
}

func TestReadWIFConfig_Complete(t *testing.T) {
	clearWIFEnv(t)
	t.Setenv("ANTHROPIC_FEDERATION_RULE_ID", "rule-1")
	t.Setenv("ANTHROPIC_ORGANIZATION_ID", "org-1")
	t.Setenv("ANTHROPIC_SERVICE_ACCOUNT_ID", "svc-1")
	t.Setenv("TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC", "tok")

	cfg, err := ReadWIFConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if cfg.FederationRuleID != "rule-1" || cfg.OrganizationID != "org-1" || cfg.ServiceAccountID != "svc-1" {
		t.Errorf("unexpected config values: %+v", cfg)
	}
}

func TestReadWIFConfig_Partial(t *testing.T) {
	clearWIFEnv(t)
	t.Setenv("ANTHROPIC_FEDERATION_RULE_ID", "rule-1")

	_, err := ReadWIFConfig()
	if err == nil {
		t.Fatal("expected error for partial config")
	}
}

func TestMintToken_NilConfig(t *testing.T) {
	_, err := MintToken(context.Background(), nil, "wrkspc_123")
	if err == nil {
		t.Fatal("expected error when cfg is nil")
	}
}

func TestMintToken_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "at_test",
			"expires_in":   3600,
		})
	}))
	defer srv.Close()

	orig := anthropicTokenURL
	anthropicTokenURL = srv.URL
	defer func() { anthropicTokenURL = orig }()

	cfg := &WIFConfig{
		FederationRuleID: "rule-1",
		OrganizationID:   "org-1",
		ServiceAccountID: "svc-1",
		jwt:              fakeJWT,
	}

	tok, err := MintToken(context.Background(), cfg, "wrkspc_123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.AccessToken != "at_test" {
		t.Errorf("expected at_test, got %s", tok.AccessToken)
	}
	if tok.ExpiresAt.Before(time.Now().Add(59 * time.Minute)) {
		t.Errorf("expiry looks too early: %v", tok.ExpiresAt)
	}
}

func TestMintToken_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
	}))
	defer srv.Close()

	orig := anthropicTokenURL
	anthropicTokenURL = srv.URL
	defer func() { anthropicTokenURL = orig }()

	cfg := &WIFConfig{jwt: fakeJWT}
	_, err := MintToken(context.Background(), cfg, "wrkspc_123")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

// helpers

func clearWIFEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"ANTHROPIC_FEDERATION_RULE_ID",
		"ANTHROPIC_ORGANIZATION_ID",
		"ANTHROPIC_SERVICE_ACCOUNT_ID",
		"TFC_WORKLOAD_IDENTITY_TOKEN_ANTHROPIC",
	} {
		os.Unsetenv(k)
	}
}

// fakeJWT is a minimal three-part JWT-shaped token.
// header.payload("{"sub":"test","aud":"anthropic"}").sig
const fakeJWT = "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJ0ZXN0IiwiYXVkIjoiYW50aHJvcGljIn0.sig"

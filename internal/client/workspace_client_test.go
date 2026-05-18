package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/auth"
)

func TestWorkspaceClient_ResolveByName_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(auth.HeaderAPIKey) != "key-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "tosspaper"},
				{"id": "wrkspc_def", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key-123"})
	id, err := c.ResolveByName(context.Background(), "tosspaper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_abc" {
		t.Errorf("expected wrkspc_abc, got %s", id)
	}
}

func TestWorkspaceClient_ResolveByName_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key-123"})
	_, err := c.ResolveByName(context.Background(), "tosspaper")
	if err == nil {
		t.Fatal("expected error when workspace not found")
	}
}

func TestWorkspaceClient_ResolveByName_DefaultWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_default", "name": ""},
			},
		})
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key-123"})
	id, err := c.ResolveByName(context.Background(), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_default" {
		t.Errorf("expected wrkspc_default, got %s", id)
	}
}

func TestWorkspaceClient_ResolveByName_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "bad-key"})
	_, err := c.ResolveByName(context.Background(), "tosspaper")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

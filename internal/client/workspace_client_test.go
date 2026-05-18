package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic/internal/auth"
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

func TestWorkspaceClient_ResolveByName_Paginated(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		callCount++
		if r.URL.Query().Get("after_id") == "" {
			json.NewEncoder(w).Encode(map[string]any{
				"data":     []map[string]string{{"id": "wrkspc_p1a", "name": "alpha"}, {"id": "wrkspc_p1b", "name": "beta"}},
				"has_more": true,
				"last_id":  "wrkspc_p1b",
			})
		} else {
			json.NewEncoder(w).Encode(map[string]any{
				"data":     []map[string]string{{"id": "wrkspc_p2a", "name": "gamma"}},
				"has_more": false,
				"last_id":  "wrkspc_p2a",
			})
		}
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key-123"})
	id, err := c.ResolveByName(context.Background(), "gamma")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_p2a" {
		t.Errorf("expected wrkspc_p2a, got %s", id)
	}
	if callCount != 2 {
		t.Errorf("expected 2 API calls for 2 pages, got %d", callCount)
	}
}

func TestWorkspaceClient_ResolveByName_EmptyLastIDGuard(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(auth.HeaderContentType, auth.MIMEApplicationJSON)
		callCount++
		json.NewEncoder(w).Encode(map[string]any{
			"data":     []map[string]string{{"id": "wrkspc_abc", "name": "alpha"}},
			"has_more": true,
			"last_id":  "",
		})
	}))
	defer srv.Close()

	orig := auth.BaseURL
	auth.BaseURL = srv.URL
	defer func() { auth.BaseURL = orig }()

	c := NewWorkspaceClient(auth.AdminAPIKey{Key: "key-123"})
	id, err := c.ResolveByName(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_abc" {
		t.Errorf("expected wrkspc_abc, got %s", id)
	}
	if callCount != 1 {
		t.Errorf("expected exactly 1 API call (guard fired), got %d", callCount)
	}
}

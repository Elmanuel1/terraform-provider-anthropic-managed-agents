package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveWorkspaceID_Found(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "key-123" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "tosspaper"},
				{"id": "wrkspc_def", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	id, err := ResolveWorkspaceID(context.Background(), AdminAPIKey{Key: "key-123"}, "tosspaper")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_abc" {
		t.Errorf("expected wrkspc_abc, got %s", id)
	}
}

func TestResolveWorkspaceID_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_abc", "name": "other"},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	_, err := ResolveWorkspaceID(context.Background(), AdminAPIKey{Key: "key-123"}, "tosspaper")
	if err == nil {
		t.Fatal("expected error when workspace not found")
	}
}

func TestResolveWorkspaceID_DefaultWorkspace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(HeaderContentType, MIMEApplicationJSON)
		json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]string{
				{"id": "wrkspc_default", "name": ""},
			},
		})
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	id, err := ResolveWorkspaceID(context.Background(), AdminAPIKey{Key: "key-123"}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "wrkspc_default" {
		t.Errorf("expected wrkspc_default, got %s", id)
	}
}

func TestResolveWorkspaceID_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()

	orig := anthropicWorkspacesURL
	anthropicWorkspacesURL = srv.URL
	defer func() { anthropicWorkspacesURL = orig }()

	_, err := ResolveWorkspaceID(context.Background(), AdminAPIKey{Key: "bad-key"}, "tosspaper")
	if err == nil {
		t.Fatal("expected error for non-200 response")
	}
}

func TestWIFBearer_NilConfig(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	err := WIFBearer{Config: nil}.Authenticate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nil WIF config")
	}
}

func TestAdminAPIKey_EmptyKey(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	err := AdminAPIKey{Key: ""}.Authenticate(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestAdminAPIKey_SetsHeaders(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	if err := (AdminAPIKey{Key: "test-key"}).Authenticate(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if req.Header.Get(HeaderAPIKey) != "test-key" {
		t.Errorf("%s not set correctly", HeaderAPIKey)
	}
	if req.Header.Get(HeaderVersion) == "" {
		t.Errorf("%s header missing", HeaderVersion)
	}
}

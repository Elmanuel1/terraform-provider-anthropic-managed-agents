package auth

// BaseURL is the Anthropic API base URL. A var so tests can point at an httptest.Server.
var BaseURL = "https://api.anthropic.com"

const (
	// Header values
	MIMEApplicationJSON = "application/json"

	// Header names
	HeaderAPIKey      = "x-api-key"
	HeaderVersion     = "anthropic-version"
	HeaderBeta        = "anthropic-beta"
	HeaderAuth        = "Authorization"
	HeaderContentType = "Content-Type"

	// API version and beta values
	APIVersion        = "2023-06-01"
	AdminBeta         = "admin-api-2025-05-21"
	AgentsBeta        = "managed-agents-2026-04-01"

)

package auth

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

	// Base URL
	BaseURL = "https://api.anthropic.com"
)

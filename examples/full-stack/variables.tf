variable "workspace_api_key" {
  description = "Anthropic workspace API key (sk-ant-api03-...)"
  type        = string
  sensitive   = true
}

variable "admin_api_key" {
  description = "Anthropic admin API key (sk-ant-admin-...) — required for workspace management"
  type        = string
  sensitive   = true
}

terraform {
  required_version = ">= 1.5.0"
  required_providers {
    anthropic = {
      source = "Elmanuel1/anthropic"
    }
  }
}

variable "federation_rule_id" { default = "" }
variable "organization_id"    { default = "" }
variable "service_account_id" { default = "" }

provider "anthropic" {
  # Admin key from env (ANTHROPIC_ADMIN_API_KEY) for workspace creation.
  # WIF credentials for agent creation.
  federation_rule_id = var.federation_rule_id
  organization_id    = var.organization_id
  service_account_id = var.service_account_id
}

resource "anthropic_workspace" "test" {
  name = "tf-wif-test"
}

resource "anthropic_agent" "test" {
  workspace_id = anthropic_workspace.test.id
  name         = "wif-test-agent"
  model        = "claude-haiku-4-5"
  model_speed  = "standard"
}

output "workspace_id" { value = anthropic_workspace.test.id }
output "agent_id"     { value = anthropic_agent.test.id }

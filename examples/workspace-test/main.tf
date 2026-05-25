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
  federation_rule_id = var.federation_rule_id
  organization_id    = var.organization_id
  service_account_id = var.service_account_id
}

resource "anthropic_workspace" "test" {
  name = "tf-wif-test-workspace"
}

output "workspace_id" {
  value = anthropic_workspace.test.id
}

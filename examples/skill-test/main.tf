terraform {
  required_version = ">= 1.5.0"
  required_providers {
    anthropic = {
      source = "Elmanuel1/anthropic"
    }
  }
}

provider "anthropic" {}

resource "anthropic_skill" "hello" {
  display_title = "Hello Skill"
  source_dir    = "${path.module}/skills/hello-skill"
}

output "skill_id" {
  value = anthropic_skill.hello.id
}

output "skill_source_hash" {
  value = anthropic_skill.hello.source_hash
}

output "skill_created_at" {
  value = anthropic_skill.hello.created_at
}

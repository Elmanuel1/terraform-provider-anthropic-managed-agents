terraform {
  required_version = ">= 1.5.0"
  required_providers {
    anthropic = {
      source  = "Elmanuel1/anthropic"
      version = "~> 0.3"
    }
  }
}

# ---------------------------------------------------------------------------
# Provider — workspace API key auth (no WIF needed for local testing)
# ---------------------------------------------------------------------------
provider "anthropic" {
  workspace_api_key = var.workspace_api_key
  admin_api_key     = var.admin_api_key
}

# ---------------------------------------------------------------------------
# Workspace (requires admin_api_key)
# ---------------------------------------------------------------------------
resource "anthropic_workspace" "test" {
  name = "test-workspace"
}

# ---------------------------------------------------------------------------
# Environment — limited networking + packages + MCP server access
# ---------------------------------------------------------------------------
resource "anthropic_environment" "test" {
  workspace_id = anthropic_workspace.test.id
  name         = "python-mcp-env"
  description  = "Test environment: limited networking, pre-installed packages, MCP access"

  networking_type        = "limited"
  allowed_hosts          = [
    "pypi.org",
    "files.pythonhosted.org",
    "registry.npmjs.org",
  ]
  allow_mcp_servers      = true
  allow_package_managers = true

  packages = jsonencode({
    pip = ["pandas", "numpy", "requests", "httpx"]
    npm = ["axios"]
  })

  metadata = {
    team = "platform"
    env  = "test"
  }
}

# ---------------------------------------------------------------------------
# Agent — MCP servers + tool permissions
# ---------------------------------------------------------------------------
resource "anthropic_agent" "test" {
  workspace_id = anthropic_workspace.test.id
  name         = "test-agent"
  model        = "claude-sonnet-4-6"
  model_speed  = "standard"
  description  = "Test agent exercising MCP servers and tool permissions"

  system = "You are a helpful assistant with access to ERP and document tools."

  # MCP servers the agent can connect to
  mcp_servers = jsonencode([
    {
      name = "erp-server"
      type = "url"
      url  = "https://erp.example.com/mcp"
    },
    {
      name = "docs-server"
      type = "url"
      url  = "https://docs.example.com/mcp"
    }
  ])

  # Tool permissions — agent_toolset grants access to all built-in tools;
  # mcp_toolset restricts which tools are exposed from each MCP server
  tools = jsonencode([
    {
      type = "agent_toolset_20260401"
    },
    {
      type            = "mcp_toolset"
      mcp_server_name = "erp-server"
      allowed_tools   = ["create_purchase_order", "list_vendors", "get_invoice"]
    },
    {
      type            = "mcp_toolset"
      mcp_server_name = "docs-server"
    }
  ])

  metadata = {
    team = "platform"
    env  = "test"
  }
}

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------
output "workspace_id" {
  value = anthropic_workspace.test.id
}

output "environment_id" {
  value = anthropic_environment.test.id
}

output "agent_id" {
  value = anthropic_agent.test.id
}

package provider

import (
	"encoding/json"
	"testing"

	"github.com/Elmanuel1/terraform-provider-anthropic-managed-agents/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullableString_Nil(t *testing.T) {
	result := nullableString(nil)
	if !result.IsNull() {
		t.Errorf("expected null, got %v", result)
	}
}

func TestNullableString_Value(t *testing.T) {
	s := "hello"
	result := nullableString(&s)
	if result.IsNull() {
		t.Fatal("expected non-null")
	}
	if result.ValueString() != "hello" {
		t.Errorf("expected hello, got %s", result.ValueString())
	}
}

func TestNullableBool_Nil(t *testing.T) {
	result := nullableBool(nil)
	if result.IsNull() {
		t.Error("expected false (not null) for nil bool")
	}
	if result.ValueBool() {
		t.Error("expected false for nil bool")
	}
}

func TestNullableBool_True(t *testing.T) {
	b := true
	result := nullableBool(&b)
	if !result.ValueBool() {
		t.Error("expected true")
	}
}

func TestNullableBool_False(t *testing.T) {
	b := false
	result := nullableBool(&b)
	if result.ValueBool() {
		t.Error("expected false")
	}
}

func TestEnvironmentModel_Fill_NoConfig(t *testing.T) {
	var m EnvironmentModel
	m.fill(client.EnvironmentResponse{
		ID:        "env_1",
		Name:      "test",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-02T00:00:00Z",
	})

	if m.Id.ValueString() != "env_1" {
		t.Errorf("unexpected id: %s", m.Id.ValueString())
	}
	if m.NetworkingType.ValueString() != "unrestricted" {
		t.Errorf("expected unrestricted, got %s", m.NetworkingType.ValueString())
	}
	if m.AllowMCPServers.ValueBool() {
		t.Error("expected AllowMCPServers=false")
	}
	if m.AllowPackageManagers.ValueBool() {
		t.Error("expected AllowPackageManagers=false")
	}
	if m.AllowedHosts.IsNull() || m.AllowedHosts.Elements() == nil {
		t.Error("expected empty (non-null) allowed_hosts list")
	}
	if !m.Packages.IsNull() {
		t.Errorf("expected null packages, got %s", m.Packages.ValueString())
	}
}

func TestEnvironmentModel_Fill_LimitedNetworking(t *testing.T) {
	trueVal := true
	var m EnvironmentModel
	m.fill(client.EnvironmentResponse{
		ID:        "env_2",
		Name:      "limited",
		CreatedAt: "2024-01-01T00:00:00Z",
		UpdatedAt: "2024-01-01T00:00:00Z",
		Config: &struct {
			Packages   json.RawMessage `json:"packages"`
			Networking *struct {
				Type                 string   `json:"type"`
				AllowedHosts         []string `json:"allowed_hosts"`
				AllowMCPServers      *bool    `json:"allow_mcp_servers"`
				AllowPackageManagers *bool    `json:"allow_package_managers"`
			} `json:"networking"`
		}{
			Networking: &struct {
				Type                 string   `json:"type"`
				AllowedHosts         []string `json:"allowed_hosts"`
				AllowMCPServers      *bool    `json:"allow_mcp_servers"`
				AllowPackageManagers *bool    `json:"allow_package_managers"`
			}{
				Type:                 "limited",
				AllowedHosts:         []string{"api.example.com"},
				AllowMCPServers:      &trueVal,
				AllowPackageManagers: nil,
			},
		},
	})

	if m.NetworkingType.ValueString() != "limited" {
		t.Errorf("expected limited, got %s", m.NetworkingType.ValueString())
	}
	if !m.AllowMCPServers.ValueBool() {
		t.Error("expected AllowMCPServers=true")
	}
	if m.AllowPackageManagers.ValueBool() {
		t.Error("expected AllowPackageManagers=false")
	}
	if m.AllowedHosts.Elements() == nil || len(m.AllowedHosts.Elements()) != 1 {
		t.Errorf("expected 1 allowed host, got %v", m.AllowedHosts.Elements())
	}
}

func TestBuildAgentBody_MinimalFields(t *testing.T) {
	data := AgentModel{
		Name:       types.StringValue("my-agent"),
		Model:      types.StringValue("claude-sonnet-4-6"),
		ModelSpeed: types.StringValue("standard"),
		System:     types.StringNull(),
		Description: types.StringNull(),
		Tools:      types.StringNull(),
	}

	body, err := buildAgentBody(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["name"] != "my-agent" {
		t.Errorf("unexpected name: %v", body["name"])
	}
	m, ok := body["model"].(map[string]string)
	if !ok {
		t.Fatalf("model is not map[string]string: %T", body["model"])
	}
	if m["id"] != "claude-sonnet-4-6" || m["speed"] != "standard" {
		t.Errorf("unexpected model: %v", m)
	}
	if _, exists := body["system"]; exists {
		t.Error("system should be absent when null")
	}
	if _, exists := body["description"]; exists {
		t.Error("description should be absent when null")
	}
	if _, exists := body["tools"]; exists {
		t.Error("tools should be absent when null")
	}
}

func TestBuildAgentBody_AllFields(t *testing.T) {
	data := AgentModel{
		Name:        types.StringValue("my-agent"),
		Model:       types.StringValue("claude-opus-4-7"),
		ModelSpeed:  types.StringValue("fast"),
		System:      types.StringValue("you are helpful"),
		Description: types.StringValue("desc"),
		Tools:       types.StringValue(`[{"type":"agent_toolset_20260401"}]`),
	}

	body, err := buildAgentBody(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if body["system"] != "you are helpful" {
		t.Errorf("unexpected system: %v", body["system"])
	}
	if body["description"] != "desc" {
		t.Errorf("unexpected description: %v", body["description"])
	}
	tools, ok := body["tools"].([]interface{})
	if !ok || len(tools) != 1 {
		t.Errorf("expected 1 tool, got %v", body["tools"])
	}
}

func TestBuildAgentBody_EmptyToolsArrayIncluded(t *testing.T) {
	data := AgentModel{
		Name:       types.StringValue("a"),
		Model:      types.StringValue("claude-sonnet-4-6"),
		ModelSpeed: types.StringValue("standard"),
		System:     types.StringNull(),
		Description: types.StringNull(),
		Tools:      types.StringValue("[]"),
	}

	body, err := buildAgentBody(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tools, exists := body["tools"]
	if !exists {
		t.Fatal("empty tools array must be included in body so callers can clear the field")
	}
	if arr, ok := tools.([]interface{}); !ok || len(arr) != 0 {
		t.Errorf("expected empty slice, got %v", tools)
	}
}

func TestAgentModel_Fill(t *testing.T) {
	desc := "an agent"
	var m AgentModel
	m.fill(client.AgentResponse{
		ID:          "agent_1",
		Name:        "my-agent",
		System:      nil,
		Description: &desc,
		Version:     3,
		CreatedAt:   "2024-01-01T00:00:00Z",
		UpdatedAt:   "2024-01-02T00:00:00Z",
		Model: struct {
			ID    string `json:"id"`
			Speed string `json:"speed"`
		}{ID: "claude-sonnet-4-6", Speed: "standard"},
	})

	if m.Id.ValueString() != "agent_1" {
		t.Errorf("unexpected id: %s", m.Id.ValueString())
	}
	if m.Model.ValueString() != "claude-sonnet-4-6" {
		t.Errorf("unexpected model: %s", m.Model.ValueString())
	}
	if m.System != (types.String{}) && !m.System.IsNull() {
		// system was nil in response — should be null
	}
	if m.Description.ValueString() != "an agent" {
		t.Errorf("unexpected description: %s", m.Description.ValueString())
	}
	if m.Version.ValueInt64() != 3 {
		t.Errorf("expected version 3, got %d", m.Version.ValueInt64())
	}
	if m.Tools.ValueString() != "[]" {
		t.Errorf("expected empty tools array, got %s", m.Tools.ValueString())
	}
}

package provider

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// unknownOnUpdateInt64 marks an Int64 field as unknown during updates so Terraform
// accepts whatever the API returns (e.g. an auto-incremented version number).
type unknownOnUpdateInt64 struct{}

func (unknownOnUpdateInt64) Description(_ context.Context) string {
	return "Value is unknown after update — the API changes it on every write."
}
func (unknownOnUpdateInt64) MarkdownDescription(ctx context.Context) string {
	return "Value is unknown after update — the API changes it on every write."
}
func (unknownOnUpdateInt64) PlanModifyInt64(_ context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() {
		return // create: leave as computed unknown
	}
	resp.PlanValue = types.Int64Unknown()
}

// unknownOnUpdateString marks a String field as unknown during updates so Terraform
// accepts whatever the API returns (e.g. an auto-updated timestamp).
type unknownOnUpdateString struct{}

func (unknownOnUpdateString) Description(_ context.Context) string {
	return "Value is unknown after update — the API changes it on every write."
}
func (unknownOnUpdateString) MarkdownDescription(ctx context.Context) string {
	return "Value is unknown after update — the API changes it on every write."
}
func (unknownOnUpdateString) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() {
		return // create: leave as computed unknown
	}
	resp.PlanValue = types.StringUnknown()
}

// apiInjectedToolKeys are fields the API adds to tool objects that users never specify.
// Stripping them keeps the stored JSON consistent with the plan value.
var apiInjectedToolKeys = map[string]bool{
	"configs":        true,
	"default_config": true,
}

// marshalJSONList serializes a list of raw JSON items back to a JSON array string,
// stripping any API-injected keys from each object so the stored value stays stable.
func marshalJSONList(raw []json.RawMessage) types.String {
	if len(raw) == 0 {
		return types.StringValue("[]")
	}
	normalized := make([]json.RawMessage, 0, len(raw))
	for _, item := range raw {
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(item, &obj); err != nil {
			normalized = append(normalized, item)
			continue
		}
		for k := range apiInjectedToolKeys {
			delete(obj, k)
		}
		b, err := json.Marshal(obj)
		if err != nil {
			normalized = append(normalized, item)
			continue
		}
		normalized = append(normalized, json.RawMessage(b))
	}
	b, err := json.Marshal(normalized)
	if err != nil {
		return types.StringValue("[]")
	}
	return types.StringValue(string(b))
}

// normalizePackages converts the API packages response (which includes a "type" key
// and empty manager arrays) to the sparse user-facing format, returning null when empty.
func normalizePackages(raw json.RawMessage) types.String {
	if len(raw) == 0 || string(raw) == "null" {
		return types.StringNull()
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return types.StringValue(string(raw))
	}
	delete(m, "type")
	for k, v := range m {
		var arr []json.RawMessage
		if json.Unmarshal(v, &arr) == nil && len(arr) == 0 {
			delete(m, k)
		}
	}
	if len(m) == 0 {
		return types.StringNull()
	}
	b, _ := json.Marshal(m)
	return types.StringValue(string(b))
}

func nullableString(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

func fillMetadata(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	vals := make(map[string]attr.Value, len(m))
	for k, v := range m {
		vals[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, vals)
}

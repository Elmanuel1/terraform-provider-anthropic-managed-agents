package provider

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// apiInjectedToolKeys are fields the API adds to tool objects that users never specify.
// Stripping them keeps the stored JSON consistent with the plan value.
var apiInjectedToolKeys = map[string]bool{
	"configs":        true,
	"default_config": true,
}

// marshalJSONList serializes a list of raw JSON items back to a JSON array string,
// stripping any API-injected keys from each object so the stored value stays stable.
// jsontypes.Normalized is used so key-ordering differences do not cause plan diffs.
func marshalJSONList(raw []json.RawMessage) (jsontypes.Normalized, error) {
	if len(raw) == 0 {
		return jsontypes.NewNormalizedValue("[]"), nil
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
		return jsontypes.Normalized{}, fmt.Errorf("marshaling JSON list: %w", err)
	}
	return jsontypes.NewNormalizedValue(string(b)), nil
}

// normalizePackages converts the API packages response (which includes a "type" key
// and empty manager arrays) to the sparse user-facing format, returning null when empty.
// jsontypes.Normalized is used so key-ordering differences do not cause plan diffs.
func normalizePackages(raw json.RawMessage) (jsontypes.Normalized, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return jsontypes.NewNormalizedNull(), nil
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		return jsontypes.Normalized{}, fmt.Errorf("parsing packages: %w", err)
	}
	delete(m, "type")
	for k, v := range m {
		var arr []json.RawMessage
		if json.Unmarshal(v, &arr) == nil && len(arr) == 0 {
			delete(m, k)
		}
	}
	if len(m) == 0 {
		return jsontypes.NewNormalizedNull(), nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return jsontypes.Normalized{}, fmt.Errorf("marshaling packages: %w", err)
	}
	return jsontypes.NewNormalizedValue(string(b)), nil
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

package provider

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func marshalJSONList(raw []json.RawMessage) types.String {
	if len(raw) == 0 {
		return types.StringValue("[]")
	}
	b, _ := json.Marshal(raw)
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

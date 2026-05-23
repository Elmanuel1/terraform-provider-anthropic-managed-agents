package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// JSONSubsetType is a custom JSON string type whose semantic equality treats
// the plan value as a structural subset of the state value. Extra keys present
// in state but absent from the plan (API-injected defaults) are ignored.
type JSONSubsetType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = JSONSubsetType{}

func (t JSONSubsetType) String() string { return "JSONSubsetType" }

func (t JSONSubsetType) ValueType(_ context.Context) attr.Value { return JSONSubsetValue{} }

func (t JSONSubsetType) Equal(o attr.Type) bool {
	_, ok := o.(JSONSubsetType)
	return ok
}

func (t JSONSubsetType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return JSONSubsetValue{StringValue: in}, nil
}

func (t JSONSubsetType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
	attrValue, err := t.StringType.ValueFromTerraform(ctx, in)
	if err != nil {
		return nil, err
	}
	sv, ok := attrValue.(basetypes.StringValue)
	if !ok {
		return nil, fmt.Errorf("unexpected value type %T", attrValue)
	}
	v, diags := t.ValueFromString(ctx, sv)
	if diags.HasError() {
		return nil, fmt.Errorf("converting StringValue: %v", diags)
	}
	return v, nil
}

// JSONSubsetValue is the value type for JSONSubsetType.
type JSONSubsetValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = JSONSubsetValue{}
	_ basetypes.StringValuableWithSemanticEquals = JSONSubsetValue{}
)

func (v JSONSubsetValue) Type(_ context.Context) attr.Type { return JSONSubsetType{} }

func (v JSONSubsetValue) Equal(o attr.Value) bool {
	other, ok := o.(JSONSubsetValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true when every key/element present in the plan
// value (v) also exists in the state value (prior) with an equal value.
// Keys present only in state are ignored — they are API-injected defaults.
func (v JSONSubsetValue) StringSemanticEquals(_ context.Context, prior basetypes.StringValuable) (bool, diag.Diagnostics) {
	priorVal, ok := prior.(JSONSubsetValue)
	if !ok {
		return false, nil
	}
	planStr := v.ValueString()
	stateStr := priorVal.ValueString()
	if planStr == stateStr {
		return true, nil
	}
	var plan, state any
	if err := json.Unmarshal([]byte(planStr), &plan); err != nil {
		return false, nil
	}
	if err := json.Unmarshal([]byte(stateStr), &state); err != nil {
		return false, nil
	}
	return jsonPlanSubsetOfState(plan, state), nil
}

// jsonPlanSubsetOfState returns true when every key in plan matches state.
// For objects: extra state keys are ignored. For arrays: compared positionally.
func jsonPlanSubsetOfState(plan, state any) bool {
	switch p := plan.(type) {
	case map[string]any:
		s, ok := state.(map[string]any)
		if !ok {
			return false
		}
		for k, pv := range p {
			sv, exists := s[k]
			if !exists {
				return false
			}
			if !jsonPlanSubsetOfState(pv, sv) {
				return false
			}
		}
		return true
	case []any:
		s, ok := state.([]any)
		if !ok || len(p) != len(s) {
			return false
		}
		for i := range p {
			if !jsonPlanSubsetOfState(p[i], s[i]) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(plan, state)
	}
}

func NewJSONSubsetValue(value string) JSONSubsetValue {
	return JSONSubsetValue{StringValue: basetypes.NewStringValue(value)}
}

func NewJSONSubsetNull() JSONSubsetValue {
	return JSONSubsetValue{StringValue: basetypes.NewStringNull()}
}

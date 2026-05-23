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

// PackagesType is a custom JSON string type whose semantic equality strips
// API-injected fields (type, empty package manager arrays) before comparing,
// so the user can write a sparse config like {"pip":["requests"]} without
// getting a perpetual diff from the API's full response.
type PackagesType struct {
	basetypes.StringType
}

var _ basetypes.StringTypable = PackagesType{}

func (t PackagesType) String() string { return "PackagesType" }

func (t PackagesType) ValueType(_ context.Context) attr.Value { return PackagesValue{} }

func (t PackagesType) Equal(o attr.Type) bool {
	_, ok := o.(PackagesType)
	return ok
}

func (t PackagesType) ValueFromString(_ context.Context, in basetypes.StringValue) (basetypes.StringValuable, diag.Diagnostics) {
	return PackagesValue{StringValue: in}, nil
}

func (t PackagesType) ValueFromTerraform(ctx context.Context, in tftypes.Value) (attr.Value, error) {
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

// PackagesValue is the value type for PackagesType.
type PackagesValue struct {
	basetypes.StringValue
}

var (
	_ basetypes.StringValuable                   = PackagesValue{}
	_ basetypes.StringValuableWithSemanticEquals = PackagesValue{}
)

func (v PackagesValue) Type(_ context.Context) attr.Type { return PackagesType{} }

func (v PackagesValue) Equal(o attr.Value) bool {
	other, ok := o.(PackagesValue)
	if !ok {
		return false
	}
	return v.StringValue.Equal(other.StringValue)
}

// StringSemanticEquals returns true when the plan value (v) matches the state
// value (prior) after stripping API-injected keys: the "type" field and any
// package-manager key whose value is an empty array.
func (v PackagesValue) StringSemanticEquals(_ context.Context, prior basetypes.StringValuable) (bool, diag.Diagnostics) {
	priorPkg, ok := prior.(PackagesValue)
	if !ok {
		return false, nil
	}
	planStripped := stripPackagesDefaults(v.ValueString())
	stateStripped := stripPackagesDefaults(priorPkg.ValueString())
	return reflect.DeepEqual(planStripped, stateStripped), nil
}

// stripPackagesDefaults removes "type" and empty-array keys from a packages
// JSON object, returning the canonical sparse map the user would write.
func stripPackagesDefaults(raw string) map[string][]string {
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	delete(m, "type")
	result := make(map[string][]string)
	for k, v := range m {
		var arr []string
		if err := json.Unmarshal(v, &arr); err == nil && len(arr) > 0 {
			result[k] = arr
		}
	}
	return result
}

func NewPackagesValue(value string) PackagesValue {
	return PackagesValue{StringValue: basetypes.NewStringValue(value)}
}

func NewPackagesNull() PackagesValue {
	return PackagesValue{StringValue: basetypes.NewStringNull()}
}

package utils

import (
	"reflect"
	"testing"

	"github.com/zclconf/go-cty/cty"
)

// TestConvertCtyToFieldType_NumericTypes tests all numeric type conversions
func TestConvertCtyToFieldType_NumericTypes(t *testing.T) {
	tests := []struct {
		name       string
		ctyVal     cty.Value
		targetType reflect.Type
		want       interface{}
	}{
		// Integer types
		{"int", cty.NumberIntVal(42), reflect.TypeOf(int(0)), int(42)},
		{"int8", cty.NumberIntVal(127), reflect.TypeOf(int8(0)), int8(127)},
		{"int16", cty.NumberIntVal(32767), reflect.TypeOf(int16(0)), int16(32767)},
		{"int32", cty.NumberIntVal(2147483647), reflect.TypeOf(int32(0)), int32(2147483647)},
		{"int64", cty.NumberIntVal(9223372036854775807), reflect.TypeOf(int64(0)), int64(9223372036854775807)},

		// Unsigned integer types
		{"uint", cty.NumberIntVal(42), reflect.TypeOf(uint(0)), uint(42)},
		{"uint8", cty.NumberIntVal(255), reflect.TypeOf(uint8(0)), uint8(255)},
		{"uint16", cty.NumberIntVal(65535), reflect.TypeOf(uint16(0)), uint16(65535)},
		{"uint32", cty.NumberIntVal(4294967295), reflect.TypeOf(uint32(0)), uint32(4294967295)},
		{"uint64", cty.NumberIntVal(9223372036854775807), reflect.TypeOf(uint64(0)), uint64(9223372036854775807)},

		// Float types
		{"float32", cty.NumberFloatVal(3.14), reflect.TypeOf(float32(0)), float32(3.14)},
		{"float64", cty.NumberFloatVal(3.14159265359), reflect.TypeOf(float64(0)), float64(3.14159265359)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(tt.ctyVal, tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCtyToFieldType() = %v (type %T), want %v (type %T)", got, got, tt.want, tt.want)
			}
		})
	}
}

// TestConvertCtyToFieldType_PrimitiveTypes tests primitive type conversions
func TestConvertCtyToFieldType_PrimitiveTypes(t *testing.T) {
	tests := []struct {
		name       string
		ctyVal     cty.Value
		targetType reflect.Type
		want       interface{}
	}{
		{"string", cty.StringVal("hello"), reflect.TypeOf(""), "hello"},
		{"bool_true", cty.BoolVal(true), reflect.TypeOf(false), true},
		{"bool_false", cty.BoolVal(false), reflect.TypeOf(false), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(tt.ctyVal, tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCtyToFieldType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConvertCtyToFieldType_NumberToString tests number to string conversion
func TestConvertCtyToFieldType_NumberToString(t *testing.T) {
	ctyVal := cty.NumberIntVal(12345)
	targetType := reflect.TypeOf("")

	got, err := ConvertCtyToFieldType(ctyVal, targetType)
	if err != nil {
		t.Fatalf("ConvertCtyToFieldType() error = %v", err)
	}

	want := "12345"
	if got != want {
		t.Errorf("ConvertCtyToFieldType() = %v, want %v", got, want)
	}
}

// TestConvertCtyToFieldType_MapTypes tests map type conversions
func TestConvertCtyToFieldType_MapTypes(t *testing.T) {
	tests := []struct {
		name       string
		ctyVal     cty.Value
		targetType reflect.Type
		want       interface{}
	}{
		{
			name: "map[string]string_from_object",
			ctyVal: cty.ObjectVal(map[string]cty.Value{
				"key1": cty.StringVal("value1"),
				"key2": cty.StringVal("value2"),
			}),
			targetType: reflect.TypeOf(map[string]string{}),
			want:       map[string]string{"key1": "value1", "key2": "value2"},
		},
		{
			name: "map[string]int_from_object",
			ctyVal: cty.ObjectVal(map[string]cty.Value{
				"a": cty.NumberIntVal(1),
				"b": cty.NumberIntVal(2),
			}),
			targetType: reflect.TypeOf(map[string]int{}),
			want:       map[string]int{"a": 1, "b": 2},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(tt.ctyVal, tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCtyToFieldType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConvertCtyToFieldType_SliceTypes tests slice type conversions
func TestConvertCtyToFieldType_SliceTypes(t *testing.T) {
	tests := []struct {
		name       string
		ctyVal     cty.Value
		targetType reflect.Type
		want       interface{}
	}{
		{
			name: "[]string_from_tuple",
			ctyVal: cty.TupleVal([]cty.Value{
				cty.StringVal("a"),
				cty.StringVal("b"),
				cty.StringVal("c"),
			}),
			targetType: reflect.TypeOf([]string{}),
			want:       []string{"a", "b", "c"},
		},
		{
			name: "[]int_from_tuple",
			ctyVal: cty.TupleVal([]cty.Value{
				cty.NumberIntVal(1),
				cty.NumberIntVal(2),
				cty.NumberIntVal(3),
			}),
			targetType: reflect.TypeOf([]int{}),
			want:       []int{1, 2, 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(tt.ctyVal, tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCtyToFieldType() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestConvertCtyToFieldType_NestedTypes tests nested complex type conversions
func TestConvertCtyToFieldType_NestedTypes(t *testing.T) {
	tests := []struct {
		name       string
		ctyVal     cty.Value
		targetType reflect.Type
		want       interface{}
	}{
		{
			name: "map[string][]string_from_object_with_tuples",
			ctyVal: cty.ObjectVal(map[string]cty.Value{
				"headers": cty.TupleVal([]cty.Value{
					cty.StringVal("Authorization"),
					cty.StringVal("Content-Type"),
				}),
				"methods": cty.TupleVal([]cty.Value{
					cty.StringVal("GET"),
					cty.StringVal("POST"),
				}),
			}),
			targetType: reflect.TypeOf(map[string][]string{}),
			want: map[string][]string{
				"headers": {"Authorization", "Content-Type"},
				"methods": {"GET", "POST"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(tt.ctyVal, tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertCtyToFieldType() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

// TestConvertCtyToFieldType_NullValues tests null value handling
func TestConvertCtyToFieldType_NullValues(t *testing.T) {
	tests := []struct {
		name       string
		targetType reflect.Type
		wantZero   bool
	}{
		{"null_to_string", reflect.TypeOf(""), true},
		{"null_to_int", reflect.TypeOf(int(0)), true},
		{"null_to_bool", reflect.TypeOf(false), true},
		{"null_to_slice", reflect.TypeOf([]string{}), true},
		{"null_to_map", reflect.TypeOf(map[string]string{}), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ConvertCtyToFieldType(cty.NullVal(cty.DynamicPseudoType), tt.targetType)
			if err != nil {
				t.Fatalf("ConvertCtyToFieldType() error = %v", err)
			}
			if tt.wantZero {
				want := reflect.Zero(tt.targetType).Interface()
				if !reflect.DeepEqual(got, want) {
					t.Errorf("ConvertCtyToFieldType() = %v, want zero value %v", got, want)
				}
			}
		})
	}
}

// TestConvertCtyToFieldType_OverflowDetection tests overflow scenarios
func TestConvertCtyToFieldType_OverflowDetection(t *testing.T) {
	// Test that very large number doesn't overflow uint8
	ctyVal := cty.NumberIntVal(300) // Larger than uint8 max (255)
	targetType := reflect.TypeOf(uint8(0))

	_, err := ConvertCtyToFieldType(ctyVal, targetType)
	if err == nil {
		t.Error("Expected error for overflow, got nil")
	}
}

package utils

import (
	"fmt"
	utf8 "unicode/utf8"
)

// typeSpec represents a type specification for a single struct field.
// It's a 2-element array where:
//   - [0]: struct class name (string)
//   - [1]: optional field specifications (map[string]interface{})
type typeSpec [2]interface{}

// createTypeSpec creates a typeSpec from a class name and optional field specifications.
func createTypeSpec(className string, fields ...map[string]interface{}) typeSpec {
	spec := typeSpec{className}
	if len(fields) > 0 && fields[0] != nil {
		spec[1] = fields[0]
	}
	return spec
}

// NewValue constructs a Value from a generic Go interface.
//
// This function converts Go types into the protobuf-based Value structure
// that describes interface field types for dynamic HCL unmarshaling.
//
// Conversion rules:
//
//	╔═══════════════════════════╤══════════════════════════════╗
//	║ Go type                   │ Conversion                   ║
//	╠═══════════════════════════╪══════════════════════════════╣
//	║ string                    │ ending SingleStruct value    ║
//	║ []string                  │ ending ListStruct value      ║
//	║ map[string]string         │ ending MapStruct value       ║
//	║                           │                              ║
//	║ [2]interface{}            │ SingleStruct value           ║
//	║ [][2]interface{}          │ ListStruct value             ║
//	║ map[string][2]interface{} │ MapStruct value              ║
//	║                           │                              ║
//	║ *Struct                   │ SingleStruct                 ║
//	║ []*Struct                 │ ListStruct                   ║
//	║ map[string]*Struct        │ MapStruct                    ║
//	║ map[[2]string]string      │ Map2Struct value             ║
//	╚═══════════════════════════╧══════════════════════════════╝
//
// Examples:
//   - "circle" → SingleStruct with ClassName="circle"
//   - []string{"square", "circle"} → ListStruct with two Structs
//   - map[string]string{"k1": "square"} → MapStruct
//
// Returns an error if the input type is not supported.
func NewValue(v interface{}) (*Value, error) {
	if v == nil {
		return nil, fmt.Errorf("cannot create Value from nil")
	}

	switch typedValue := v.(type) {
	case string:
		return newValueFromString(typedValue)

	case [2]interface{}:
		return newValueFromTypeSpec(typedValue)

	case []string:
		return newValueFromStringSlice(typedValue)

	case [][2]interface{}:
		return newValueFromTypeSpecSlice(typedValue)

	case map[string]string:
		return newValueFromStringMap(typedValue)

	case map[string][2]interface{}:
		return newValueFromTypeSpecMap(typedValue)

	case map[[2]string]string:
		return newValueFromString2DMap(typedValue)

	case map[[2]string][2]interface{}:
		return newValueFromTypeSpec2DMap(typedValue)

	case *Struct:
		return &Value{Kind: &Value_SingleStruct{SingleStruct: typedValue}}, nil

	case []*Struct:
		listStruct := &ListStruct{ListFields: typedValue}
		return &Value{Kind: &Value_ListStruct{ListStruct: listStruct}}, nil

	case map[string]*Struct:
		mapStruct := &MapStruct{MapFields: typedValue}
		return &Value{Kind: &Value_MapStruct{MapStruct: mapStruct}}, nil

	case map[string]*MapStruct:
		map2Struct := &Map2Struct{Map2Fields: typedValue}
		return &Value{Kind: &Value_Map2Struct{Map2Struct: map2Struct}}, nil

	default:
		return nil, fmt.Errorf("unsupported type for NewValue: %T", v)
	}
}

// newValueFromString creates a Value from a simple string class name.
func newValueFromString(className string) (*Value, error) {
	spec := createTypeSpec(className)
	return newValueFromTypeSpec(spec)
}

// newValueFromTypeSpec creates a Value from a type specification.
func newValueFromTypeSpec(spec typeSpec) (*Value, error) {
	structSpec, err := newSingleStruct(spec)
	if err != nil {
		return nil, fmt.Errorf("failed to create SingleStruct: %w", err)
	}
	return &Value{Kind: &Value_SingleStruct{SingleStruct: structSpec}}, nil
}

// newValueFromStringSlice creates a Value from a slice of string class names.
func newValueFromStringSlice(classNames []string) (*Value, error) {
	specs := make([][2]interface{}, len(classNames))
	for i, className := range classNames {
		specs[i] = createTypeSpec(className)
	}
	return newValueFromTypeSpecSlice(specs)
}

// newValueFromTypeSpecSlice creates a Value from a slice of type specifications.
func newValueFromTypeSpecSlice(specs [][2]interface{}) (*Value, error) {
	listStruct, err := newListStruct(specs)
	if err != nil {
		return nil, fmt.Errorf("failed to create ListStruct: %w", err)
	}
	return &Value{Kind: &Value_ListStruct{ListStruct: listStruct}}, nil
}

// newValueFromStringMap creates a Value from a map of string class names.
func newValueFromStringMap(classNames map[string]string) (*Value, error) {
	specs := make(map[string][2]interface{}, len(classNames))
	for key, className := range classNames {
		specs[key] = createTypeSpec(className)
	}
	return newValueFromTypeSpecMap(specs)
}

// newValueFromTypeSpecMap creates a Value from a map of type specifications.
func newValueFromTypeSpecMap(specs map[string][2]interface{}) (*Value, error) {
	mapStruct, err := newMapStruct(specs)
	if err != nil {
		return nil, fmt.Errorf("failed to create MapStruct: %w", err)
	}
	return &Value{Kind: &Value_MapStruct{MapStruct: mapStruct}}, nil
}

// newValueFromString2DMap creates a Value from a 2D map of string class names.
func newValueFromString2DMap(classNames map[[2]string]string) (*Value, error) {
	specs := make(map[[2]string][2]interface{}, len(classNames))
	for key, className := range classNames {
		specs[key] = createTypeSpec(className)
	}
	return newValueFromTypeSpec2DMap(specs)
}

// newValueFromTypeSpec2DMap creates a Value from a 2D map of type specifications.
func newValueFromTypeSpec2DMap(specs map[[2]string][2]interface{}) (*Value, error) {
	map2Struct, err := newMap2Struct(specs)
	if err != nil {
		return nil, fmt.Errorf("failed to create Map2Struct: %w", err)
	}
	return &Value{Kind: &Value_Map2Struct{Map2Struct: map2Struct}}, nil
}

// NewStruct constructs a Struct specification for dynamic type unmarshaling.
//
// A Struct describes the runtime types of interface fields in a Go struct.
// This is used during HCL/JSON unmarshaling to know which concrete types
// to instantiate for interface fields.
//
// Parameters:
//   - className: The class name (Go struct type name), must be non-empty
//   - fieldSpecs: Optional map specifying field types (field name → type specification)
//
// The map values are converted using NewValue. See NewValue for conversion rules.
//
// Examples:
//   NewStruct("geo", map[string]interface{}{
//       "Shape": "circle",  // Shape field should be a circle
//   })
//
//   NewStruct("child", map[string]interface{}{
//       "Brand": map[string][2]interface{}{
//           "abc1": {"toy", map[string]interface{}{"Geo": ...}},
//       },
//   })
//
// Returns a Struct that can be passed to UnmarshalSpec functions.
// Returns an error if className is empty or field specifications are invalid.
func NewStruct(className string, fieldSpecs ...map[string]interface{}) (*Struct, error) {
	if className == "" {
		return nil, fmt.Errorf("className cannot be empty")
	}

	structSpec := &Struct{ClassName: className}

	// No field specifications provided
	if len(fieldSpecs) == 0 || fieldSpecs[0] == nil {
		return structSpec, nil
	}

	// Convert field specifications to Values
	structSpec.Fields = make(map[string]*Value, len(fieldSpecs[0]))
	for fieldName, fieldSpec := range fieldSpecs[0] {
		if !utf8.ValidString(fieldName) {
			return nil, fmt.Errorf("field name contains invalid UTF-8: %q", fieldName)
		}

		fieldValue, err := NewValue(fieldSpec)
		if err != nil {
			return nil, fmt.Errorf("invalid specification for field %q: %w", fieldName, err)
		}

		structSpec.Fields[fieldName] = fieldValue
	}

	return structSpec, nil
}

// newSingleStruct creates a Struct from a type specification.
// The spec is a [2]interface{} where:
//   - [0]: class name (string)
//   - [1]: optional field specifications (map[string]interface{})
func newSingleStruct(spec typeSpec) (*Struct, error) {
	className, ok := spec[0].(string)
	if !ok {
		return nil, fmt.Errorf("class name must be a string, got %T", spec[0])
	}

	if spec[1] == nil {
		return NewStruct(className)
	}

	fields, ok := spec[1].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("field specifications must be map[string]interface{}, got %T", spec[1])
	}

	return NewStruct(className, fields)
}

// newListStruct creates a ListStruct from a slice of type specifications.
func newListStruct(specs [][2]interface{}) (*ListStruct, error) {
	structs := make([]*Struct, len(specs))

	for i, spec := range specs {
		structSpec, err := newSingleStruct(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid specification at index %d: %w", i, err)
		}
		structs[i] = structSpec
	}

	return &ListStruct{ListFields: structs}, nil
}

// newMapStruct creates a MapStruct from a map of type specifications.
func newMapStruct(specs map[string][2]interface{}) (*MapStruct, error) {
	structs := make(map[string]*Struct, len(specs))

	for key, spec := range specs {
		structSpec, err := newSingleStruct(spec)
		if err != nil {
			return nil, fmt.Errorf("invalid specification for key %q: %w", key, err)
		}
		structs[key] = structSpec
	}

	return &MapStruct{MapFields: structs}, nil
}

// newMap2Struct creates a Map2Struct from a 2D map of type specifications.
// This handles nested map structures where the key is a 2-element string array.
func newMap2Struct(specs map[[2]string][2]interface{}) (*Map2Struct, error) {
	// Group specifications by the first key dimension
	groupedSpecs := make(map[string]map[string][2]interface{})

	for key, spec := range specs {
		firstKey := key[0]
		secondKey := key[1]

		if groupedSpecs[firstKey] == nil {
			groupedSpecs[firstKey] = make(map[string][2]interface{})
		}
		groupedSpecs[firstKey][secondKey] = spec
	}

	// Convert grouped specifications to MapStructs
	map2Fields := make(map[string]*MapStruct, len(groupedSpecs))

	for firstKey, secondLevelSpecs := range groupedSpecs {
		mapStruct, err := newMapStruct(secondLevelSpecs)
		if err != nil {
			return nil, fmt.Errorf("invalid specification for first-level key %q: %w", firstKey, err)
		}
		map2Fields[firstKey] = mapStruct
	}

	return &Map2Struct{Map2Fields: map2Fields}, nil
}

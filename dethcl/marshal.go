// Package dethcl implements marshal and unmarshal between HCL string and go struct.
package dethcl

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Marshaler is the interface implemented by types that can marshal themselves into HCL.
// The MarshalHCL method should return the HCL encoding of the receiver.
type Marshaler interface {
	MarshalHCL() ([]byte, error)
}

// Marshal encodes a Go value into HCL format.
//
// The value can be a struct, map, slice, or any Go type with hcl struct tags.
// If the value implements Marshaler, its MarshalHCL method is called.
// Otherwise, Marshal uses reflection to encode the value.
//
// Supported types:
//   - Primitives: string, int, float, bool
//   - Structs with hcl tags
//   - Maps: map[string]T, map[[2]string]T (with labels)
//   - Slices: []T
//   - Interfaces (encoded as their concrete type)
//
// HCL struct tag modifiers:
//   - "name" - field name in HCL
//   - "name,optional" - omit if zero value
//   - "name,block" - encode as HCL block
//   - "name,label" - use as block label
//   - "-" - ignore field
//
// Example:
//
//	type Config struct {
//	    Name     string            `hcl:"name"`
//	    Services map[string]*Service `hcl:"service,block"`
//	}
//
//	cfg := &Config{Name: "app", Services: map[string]*Service{
//	    "api": {Port: 8080},
//	}}
//	hcl, err := Marshal(cfg)
//	// Output:
//	// name = "app"
//	// service "api" {
//	//   port = 8080
//	// }
//
// Returns the HCL encoding or an error if marshaling fails.
func Marshal(current any) ([]byte, error) {
	if current == nil {
		return nil, nil
	}
	return MarshalLevel(current, 0)
}

// MarshalLevel encodes a Go value into HCL format at a specific indentation level.
//
// This function is similar to Marshal but allows control over indentation depth.
// It is primarily used for encoding nested structures where the indentation level
// needs to be tracked for proper formatting.
//
// Parameters:
//   - current: the value to encode (struct, map, slice, or primitive)
//   - level: indentation level (0 for root, incremented for nested blocks)
//
// Example:
//
//	type Service struct {
//	    Port int `hcl:"port"`
//	}
//
//	svc := &Service{Port: 8080}
//	hcl, err := MarshalLevel(svc, 1)
//	// Output (with level 1 indentation):
//	// {
//	//   port = 8080
//	// }
//
// Returns the indented HCL encoding or an error if marshaling fails.
func MarshalLevel(current any, level int) ([]byte, error) {
	return marshalLevel(current, false, level)
}

// marshalLevel is the internal routing function for marshaling with control over indentation and formatting.
// It routes structs/pointers to marshal() and other types to encoding().
//
// Parameters:
//   - current: value to encode
//   - equal: whether to prefix output with "=" (for attribute-style encoding)
//   - level: indentation level
//   - keyname: optional label values for blocks
//
// Returns nil for zero values, otherwise delegates to appropriate encoding function.
func marshalLevel(current any, equal bool, level int, keyname ...string) ([]byte, error) {
	reflectValue := reflect.ValueOf(current)
	if reflectValue.IsValid() && reflectValue.IsZero() {
		return nil, nil
	}

	switch reflectValue.Kind() {
	case reflect.Pointer, reflect.Struct:
		return marshal(current, level, keyname...)
	default:
	}

	return encoding(current, equal, level, keyname...)
}

// marshal encodes a struct or pointer into HCL format with proper indentation and block structure.
// This is the core marshaling function that handles:
//   - Custom Marshaler interface implementation
//   - Primitive types (int, string, bool, float, etc.)
//   - Struct fields categorization into simple and complex fields
//   - Label extraction and formatting
//   - Nested block formatting with proper indentation
//
// The function separates fields into:
//   - Simple fields: encoded using gohcl (strings, numbers, simple slices/maps)
//   - Complex fields: encoded recursively (structs, interfaces, maps of structs)
//   - Label fields: used as block labels in HCL output
//
// Parameters:
//   - current: struct or pointer to marshal
//   - level: indentation level (0 for root)
//   - keyname: optional label values from parent context
//
// Returns formatted HCL bytes with proper indentation and block structure.
func marshal(current any, level int, keyname ...string) ([]byte, error) {
	if current == nil {
		return nil, nil
	}
	indentation := indent(level + 1)
	parentIndent := indent(level)

	if marshaler, ok := current.(Marshaler); ok {
		encoded, err := marshaler.MarshalHCL()
		if err != nil {
			return nil, err
		}

		result := string(encoded)
		if level > 0 {
			if isBlank(encoded) {
				result = "\n"
			} else {
				result = "\n" + result + "\n"
			}
		}
		result = strings.ReplaceAll(result, "\n", "\n"+parentIndent)
		if level > 0 {
			result = fmt.Sprintf("{%s%s}", indentation, result)
		}
		return []byte(result), nil
	}

	structType := reflect.TypeOf(current)
	structValue := reflect.ValueOf(current)
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
		structValue = structValue.Elem()
	}

	switch structType.Kind() {
	case reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Complex64, reflect.Complex128:
		if structValue.IsValid() {
			return []byte(fmt.Sprintf("= %v", structValue.Interface())), nil
		}
		return nil, nil
	case reflect.String:
		if structValue.IsValid() {
			return []byte(" = " + structValue.String()), nil
		}
		return nil, nil
	case reflect.Pointer:
		return marshal(structValue.Elem().Interface(), level, keyname...)
	default:
	}

	categorizedFields, err := getFields(structType, structValue)
	if err != nil {
		return nil, err
	}

	var simpleFields []reflect.StructField
	for _, marshalField := range categorizedFields {
		if !marshalField.out {
			simpleFields = append(simpleFields, marshalField.field)
		}
	}
	simpleType := reflect.StructOf(simpleFields)
	simpleStruct := reflect.New(simpleType).Elem()
	var complexFields []*marshalOut
	var labels []string

	fieldIndex := 0
	for _, marshalField := range categorizedFields {
		field := marshalField.field
		fieldValue := marshalField.value
		if marshalField.out {
			complexField, err := getOutlier(field, fieldValue, level)
			if err != nil {
				return nil, err
			}
			complexFields = append(complexFields, complexField...)
		} else {
			fieldTag := field.Tag
			tagParts := parseHCLTag(fieldTag)
			if tagParts[1] == tagModifierLabel {
				label := fieldValue.Interface().(string)
				if keyname == nil || keyname[0] != label {
					labels = append(labels, label)
				}
				fieldIndex++
				continue
			}
			simpleStruct.Field(fieldIndex).Set(fieldValue)
			fieldIndex++
		}
	}

	hclFile := hclwrite.NewEmptyFile()
	gohcl.EncodeIntoBody(simpleStruct.Addr().Interface(), hclFile.Body())
	encoded := hclFile.Bytes()

	result := string(encoded)
	result = indentation + strings.ReplaceAll(result, "\n", "\n"+indentation)

	var lines []string
	for _, item := range complexFields {
		line := string(item.b0) + " "
		if item.encode {
			line += "= "
		}
		if len(item.b1) > 0 {
			line += `"` + strings.Join(item.b1, `" "`) + `" `
		}
		line += string(item.b2)
		lines = append(lines, line)
	}
	if len(lines) > 0 {
		result += strings.Join(lines, "\n"+indentation)
	}

	result = strings.TrimRight(result, " \t\n\r")
	if level > 0 { // not root
		result = fmt.Sprintf("{\n%s\n%s}", result, parentIndent)
		if labels != nil {
			result = "\"" + strings.Join(labels, "\" \"") + "\" " + result
		}
	}

	return []byte(result), nil
}

// marshalField represents a categorized struct field during marshaling.
// Fields are categorized into simple fields (encoded by gohcl) and
// complex fields (encoded recursively with special handling).
type marshalField struct {
	field reflect.StructField // The struct field metadata
	value reflect.Value       // The field's actual value
	out   bool                // true if complex field requiring special marshaling
}

// getFields categorizes struct fields into simple and complex fields for marshaling.
// It analyzes each field to determine if it requires special handling based on:
//   - Field type (struct, interface, pointer, map, slice)
//   - Element types for collections
//   - HCL tags and modifiers
//
// Simple fields (out=false): primitives, simple slices/maps encoded by gohcl
// Complex fields (out=true): structs, interfaces, maps/slices of structs requiring recursive encoding
//
// The function also handles:
//   - Embedded/anonymous structs (recursively flattens their fields)
//   - Pointer unwrapping for maps and slices
//   - Tag-based field filtering (ignores unexported and tagged fields)
//   - Auto-tagging of untagged fields with appropriate modifiers
//
// Returns a slice of categorized marshalField instances.
func getFields(structType reflect.Type, structValue reflect.Value) ([]*marshalField, error) {
	var categorizedFields []*marshalField
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type
		if !unicode.IsUpper([]rune(field.Name)[0]) {
			continue
		}
		fieldValue := structValue.Field(i)
		tagParts := parseHCLTag(field.Tag)
		tagName := tagParts[0]
		if tagName == tagIgnore || (len(tagName) >= 2 && tagName[len(tagName)-2:] == tagIgnoreSuffix) {
			continue
		}

		if field.Anonymous && tagName == "" {
			switch fieldType.Kind() {
			case reflect.Ptr:
				embeddedFields, err := getFields(fieldType.Elem(), fieldValue.Elem())
				if err != nil {
					return nil, err
				}
				categorizedFields = append(categorizedFields, embeddedFields...)
			case reflect.Struct:
				embeddedFields, err := getFields(fieldType, fieldValue)
				if err != nil {
					return nil, err
				}
				categorizedFields = append(categorizedFields, embeddedFields...)
			default:
			}
			continue
		}

		// treat field of type pointer e.g. *map[string]*Example, the same as map[string]*Example
		if fieldType.Kind() == reflect.Pointer && (fieldType.Elem().Kind() == reflect.Slice || fieldType.Elem().Kind() == reflect.Map) {
			fieldType = fieldType.Elem()
			fieldValue = fieldValue.Elem()
			if !fieldValue.IsValid() {
				continue
			}
		}
		needsSpecialMarshaling := false
		switch fieldType.Kind() {
		case reflect.Interface, reflect.Pointer, reflect.Struct:
			needsSpecialMarshaling = true
		case reflect.Slice:
			if fieldValue.Len() == 0 {
				needsSpecialMarshaling = true
				break
			}
			switch fieldValue.Index(0).Kind() {
			case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct:
				needsSpecialMarshaling = true
			default:
			}
		case reflect.Map:
			if fieldValue.Len() == 0 {
				needsSpecialMarshaling = true
				break
			}
			switch fieldValue.MapIndex(fieldValue.MapKeys()[0]).Kind() {
			case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct:
				needsSpecialMarshaling = true
			default:
			}
		default:
			if fieldValue.IsValid() && fieldValue.IsZero() {
				continue
			}
		}
		if tagName == "" {
			if needsSpecialMarshaling {
				field.Tag = reflect.StructTag(fmt.Sprintf(`hcl:"%s,%s"`, strings.ToLower(field.Name), tagModifierBlock))
			} else {
				field.Tag = reflect.StructTag(fmt.Sprintf(`hcl:"%s,%s"`, strings.ToLower(field.Name), tagModifierOptional))
			}
		}
		categorizedFields = append(categorizedFields, &marshalField{field, fieldValue, needsSpecialMarshaling})
	}
	return categorizedFields, nil
}

// marshalOut represents the marshaled output components for a complex field.
// Complex fields are formatted as: b0 [= ] ["b1" "b1" ...] b2
// For example: "service \"api\" \"web\" { port = 8080 }"
type marshalOut struct {
	b0     []byte   // Field name/tag (e.g., "service")
	b1     []string // Labels for the block (e.g., ["api", "web"])
	b2     []byte   // Marshaled field content (e.g., "{ port = 8080 }")
	encode bool     // true if should add "=" between b0 and b2 (for attribute-style encoding)
}

// indent returns the indentation string for a given level (2 spaces per level).
func indent(level int) string {
	return strings.Repeat("  ", level)
}

// needsLoopMarshaling checks if a value requires loop-based marshaling (for structs, pointers, or interfaces containing them).
// Used to determine if slice/map elements should be marshaled individually as blocks.
func needsLoopMarshaling(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Pointer, reflect.Struct:
		return true
	case reflect.Interface:
		elem := value.Elem()
		return elem.Kind() == reflect.Pointer || elem.Kind() == reflect.Struct
	default:
		return false
	}
}

// getOutlier marshals complex fields (interfaces, structs, maps, slices) into marshalOut format.
// This function handles fields that cannot be encoded by gohcl and require recursive marshaling.
//
// It processes different types as follows:
//   - Interfaces/Pointers: Direct recursive marshaling
//   - Structs: Marshaled as nested blocks
//   - Slices: Each element marshaled separately if contains structs/interfaces, otherwise as array
//   - Maps: Each value marshaled as labeled block if contains structs/interfaces, otherwise as map
//
// Empty slices are encoded as "[]", empty maps as "{\n}".
// Blank (whitespace-only) outputs are skipped.
//
// Parameters:
//   - field: struct field metadata with HCL tags
//   - oriField: the field's value
//   - level: current indentation level
//
// Returns a slice of marshalOut components for formatting into HCL output.
func getOutlier(field reflect.StructField, oriField reflect.Value, level int) ([]*marshalOut, error) {
	var empty []*marshalOut
	fieldTag := field.Tag
	typ := field.Type
	newlevel := level + 1

	// treat ptr the same as the underlying type e.g. *Example, Example
	if typ.Kind() == reflect.Ptr && (typ.Elem().Kind() == reflect.Map || typ.Elem().Kind() == reflect.Slice) {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Interface, reflect.Pointer:
		newCurrent := oriField.Interface()
		bs, err := MarshalLevel(newCurrent, newlevel)
		if err != nil {
			return nil, err
		}
		if isBlank(bs) {
			return nil, nil
		}
		empty = append(empty, &marshalOut{extractHCLTagName(fieldTag), nil, bs, false})
	case reflect.Struct:
		var newCurrent any
		if oriField.CanAddr() {
			newCurrent = oriField.Addr().Interface()
		} else {
			newCurrent = oriField.Interface()
		}
		bs, err := MarshalLevel(newCurrent, newlevel)
		if err != nil {
			return nil, err
		}
		if isBlank(bs) {
			return nil, nil
		}
		empty = append(empty, &marshalOut{extractHCLTagName(fieldTag), nil, bs, false})
	case reflect.Slice:
		results, err := handleSlice(field, oriField, newlevel)
		if err != nil {
			return nil, err
		}
		empty = append(empty, results...)
	case reflect.Map:
		results, err := handleMap(field, oriField, level, newlevel)
		if err != nil {
			return nil, err
		}
		empty = append(empty, results...)
	default:
	}
	return empty, nil
}

func handleSlice(field reflect.StructField, oriField reflect.Value, level int) ([]*marshalOut, error) {
	if oriField.IsNil() {
		return nil, nil
	}

	n := oriField.Len()
	fieldTag := field.Tag
	if n < 1 {
		return []*marshalOut{{extractHCLTagName(fieldTag), nil, []byte(`[]`), true}}, nil
	}

	first := oriField.Index(0)
	isLoop := needsLoopMarshaling(first)

	var results []*marshalOut
	if isLoop {
		for i := 0; i < n; i++ {
			item := oriField.Index(i)
			bs, err := MarshalLevel(item.Interface(), level)
			if err != nil {
				return nil, err
			}
			if isBlank(bs) {
				continue
			}
			results = append(results, &marshalOut{extractHCLTagName(fieldTag), nil, bs, false})
		}
	} else {
		bs, err := MarshalLevel(oriField.Interface(), level)
		if err != nil {
			return nil, err
		}
		if isBlank(bs) {
			return nil, nil
		}
		results = append(results, &marshalOut{extractHCLTagName(fieldTag), nil, bs, true})
	}
	return results, nil
}

func handleMap(field reflect.StructField, oriField reflect.Value, currentLevel, level int) ([]*marshalOut, error) {
	if oriField.IsNil() {
		return nil, nil
	}

	n := oriField.Len()
	fieldTag := field.Tag
	if n < 1 {
		leading := indent(currentLevel + 1)
		return []*marshalOut{{extractHCLTagName(fieldTag), nil, []byte("{\n" + leading + "}"), false}}, nil
	}

	first := oriField.MapIndex(oriField.MapKeys()[0])
	isLoop := needsLoopMarshaling(first)
	typ := field.Type
	// treat ptr the same as the underlying type e.g. *Example, Example
	if typ.Kind() == reflect.Ptr && (typ.Elem().Kind() == reflect.Map || typ.Elem().Kind() == reflect.Slice) {
		typ = typ.Elem()
	}

	var results []*marshalOut
	if isLoop {
		iter := oriField.MapRange()
		for iter.Next() {
			k := iter.Key()
			var arr []string
			switch k.Kind() {
			case reflect.Array, reflect.Slice:
				for i := 0; i < k.Len(); i++ {
					item := k.Index(i)
					if !item.IsZero() {
						arr = append(arr, item.String())
					}
				}
			default:
				arr = append(arr, k.String())
			}

			v := iter.Value()
			var bs []byte
			var err error
			bs, err = marshal(v.Interface(), level, arr...)
			if err != nil {
				return nil, err
			}
			if isBlank(bs) {
				continue
			}
			results = append(results, &marshalOut{extractHCLTagName(fieldTag), arr, bs, false})
		}
	} else {
		bs, err := MarshalLevel(oriField.Interface(), level)
		if err != nil {
			return nil, err
		}
		if isBlank(bs) {
			return nil, nil
		}
		equal := true
		if typ.Elem().Kind() == reflect.Interface {
			equal = false
		}
		results = append(results, &marshalOut{extractHCLTagName(fieldTag), nil, bs, equal})
	}
	return results, nil
}

// isBlank checks if a byte slice contains only whitespace characters.
// Used to skip empty marshaled output (e.g., empty structs or nil values).
// Returns true if the slice contains only spaces, tabs, newlines, or carriage returns.
func isBlank(bs []byte) bool {
	for _, b := range bs {
		if b != ' ' && b != '\t' && b != '\n' && b != '\r' {
			return false
		}
	}
	return true
}

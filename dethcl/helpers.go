package dethcl

import (
	"reflect"
	"strings"
	"unicode"
)

// HCLFieldInfo contains parsed information about a struct field's HCL tags.
type HCLFieldInfo struct {
	Field    reflect.StructField
	Value    reflect.Value
	TagName  string // e.g., "name"
	Modifier string // e.g., "label", "block", "optional"
	IsLabel  bool
	IsBlock  bool
	IsIgnore bool
}

// parseFieldInfo extracts HCL tag information from a struct field.
// Returns nil if the field should be ignored (unexported or tagged with "-").
func parseFieldInfo(field reflect.StructField, value reflect.Value) *HCLFieldInfo {
	// Skip unexported fields
	if !unicode.IsUpper([]rune(field.Name)[0]) {
		return nil
	}

	tag := parseHCLTag(field.Tag)
	tagName := tag[0]
	modifier := tag[1]

	// Check for ignore tag
	if tagName == tagIgnore || (len(tagName) >= 2 && tagName[len(tagName)-2:] == ","+tagIgnore) {
		return nil
	}

	info := &HCLFieldInfo{
		Field:    field,
		Value:    value,
		TagName:  tagName,
		Modifier: modifier,
		IsLabel:  strings.ToLower(modifier) == tagModifierLabel,
		IsBlock:  strings.Contains(strings.ToLower(modifier), tagModifierBlock),
		IsIgnore: false,
	}

	return info
}

// getStructFields returns parsed field information for all fields in a struct type.
// Handles embedded structs recursively.
func getStructFields(t reflect.Type, oriValue reflect.Value) ([]*HCLFieldInfo, error) {
	var fields []*HCLFieldInfo

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := oriValue.Field(i)

		info := parseFieldInfo(field, fieldValue)
		if info == nil {
			continue
		}

		// Handle embedded/anonymous structs
		if field.Anonymous && info.TagName == "" {
			typ := field.Type
			if typ.Kind() == reflect.Ptr {
				typ = typ.Elem()
				fieldValue = fieldValue.Elem()
				if !fieldValue.IsValid() {
					continue
				}
			}

			if typ.Kind() == reflect.Struct {
				embeddedFields, err := getStructFields(typ, fieldValue)
				if err != nil {
					return nil, err
				}
				fields = append(fields, embeddedFields...)
				continue
			}
		}

		// Auto-generate tag name from field name if not specified
		if info.TagName == "" {
			info.TagName = strings.ToLower(field.Name)
		}

		fields = append(fields, info)
	}

	return fields, nil
}

// isComplexField returns true if the field requires special handling (blocks, interfaces, maps, slices of structs).
func isComplexField(fieldValue reflect.Value, fieldType reflect.Type) bool {
	// Treat pointer fields the same as their underlying type for slices/maps
	if fieldType.Kind() == reflect.Pointer && (fieldType.Elem().Kind() == reflect.Slice || fieldType.Elem().Kind() == reflect.Map) {
		fieldType = fieldType.Elem()
		fieldValue = fieldValue.Elem()
		if !fieldValue.IsValid() {
			return false
		}
	}

	switch fieldType.Kind() {
	case reflect.Interface, reflect.Pointer, reflect.Struct:
		return true
	case reflect.Slice:
		if fieldValue.Len() == 0 {
			return false
		}
		// Check if slice contains complex types
		first := fieldValue.Index(0)
		switch first.Kind() {
		case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct:
			return true
		}
	case reflect.Map:
		if fieldValue.Len() == 0 {
			return false
		}
		// Check if map values are complex types
		first := fieldValue.MapIndex(fieldValue.MapKeys()[0])
		switch first.Kind() {
		case reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice, reflect.Struct:
			return true
		}
	}
	return false
}

// filterFieldsByType separates fields into different categories for processing.
type fieldCategories struct {
	Labels  []*HCLFieldInfo // Fields marked with "label" modifier
	Simple  []*HCLFieldInfo // Simple fields that can be encoded with gohcl
	Complex []*HCLFieldInfo // Complex fields requiring special handling (blocks, interfaces, etc.)
}

// categorizeFields separates struct fields into labels, simple, and complex categories.
func categorizeFields(fields []*HCLFieldInfo) *fieldCategories {
	cats := &fieldCategories{
		Labels:  make([]*HCLFieldInfo, 0),
		Simple:  make([]*HCLFieldInfo, 0),
		Complex: make([]*HCLFieldInfo, 0),
	}

	for _, info := range fields {
		if info.IsLabel {
			cats.Labels = append(cats.Labels, info)
			continue
		}

		if isComplexField(info.Value, info.Field.Type) {
			cats.Complex = append(cats.Complex, info)
		} else {
			// Skip zero values for simple fields to avoid cluttering output
			if !info.Value.IsValid() || info.Value.IsZero() {
				continue
			}
			cats.Simple = append(cats.Simple, info)
		}
	}

	return cats
}

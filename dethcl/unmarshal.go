package dethcl

import (
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"unicode"

	"github.com/genelet/horizon/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
)

// Unmarshaler is the interface implemented by types that can unmarshal themselves from HCL.
// The UnmarshalHCL method should decode the HCL data and populate the receiver.
// Labels parameter contains HCL label values (e.g., block names) if applicable.
type Unmarshaler interface {
	UnmarshalHCL([]byte, ...string) error
}

// Unmarshal decodes HCL data into a Go value.
//
// The HCL data must be valid HCL syntax. The value pointed to by current is populated
// with the decoded data. If current implements Unmarshaler, its UnmarshalHCL method
// is called. Otherwise, Unmarshal uses reflection to populate the value.
//
// Parameters:
//   - hclData: HCL data as bytes
//   - current: pointer to struct, []interface{}, or map[string]interface{}
//   - labels: optional HCL label values (for blocks with labels)
//
// Supported types:
//   - Primitives: string, int, float, bool
//   - Collections: []T, map[string]T
//   - Structs with hcl tags
//   - map[string]interface{} and []interface{} for dynamic content
//
// Example:
//
//	type Config struct {
//	    Name string `hcl:"name"`
//	    Port int    `hcl:"port,optional"`
//	}
//
//	hcl := []byte(`name = "api"\nport = 8080`)
//	var cfg Config
//	err := Unmarshal(hcl, &cfg)
//
// Returns an error if decoding fails.
func Unmarshal(hclData []byte, current interface{}, labels ...string) error {
	if current == nil {
		return nil
	}
	rv := reflect.ValueOf(current)
	if rv.Kind() != reflect.Pointer {
		return fmt.Errorf("non-pointer or nil data")
	}
	unmarshaler, ok := current.(Unmarshaler)
	if ok {
		return unmarshaler.UnmarshalHCL(hclData, labels...)
	}
	return UnmarshalSpec(hclData, current, nil, nil, labels...)
}

// UnmarshalSpec decodes HCL data into a Go value with dynamic type resolution.
//
// This function extends Unmarshal by supporting interface fields through runtime
// type specifications. Use this when your structs contain interface fields whose
// concrete types are determined at runtime.
//
// Parameters:
//   - hclData: HCL data as bytes
//   - current: pointer to target struct
//   - spec: Struct specification describing interface field types (created with utils.NewStruct)
//   - ref: type registry mapping type names to zero-value instances
//   - labels: optional HCL label values
//
// The spec parameter describes how interface fields should be decoded:
//
//	spec, err := utils.NewStruct("Geo", map[string]interface{}{
//	    "Shape": "Circle",  // Shape interface should be decoded as Circle
//	})
//
// The ref parameter provides zero-value instances for all possible types:
//
//	ref := map[string]interface{}{
//	    "Circle": &Circle{},
//	    "Square": &Square{},
//	    "Geo":    &Geo{},
//	}
//
// Example:
//
//	type Shape interface { Area() float64 }
//	type Circle struct { Radius float64 `hcl:"radius"` }
//	type Geo struct {
//	    Name  string `hcl:"name"`
//	    Shape Shape  `hcl:"shape,block"`
//	}
//
//	hcl := []byte(`name = "test"\nshape { radius = 5.0 }`)
//	spec, _ := utils.NewStruct("Geo", map[string]interface{}{"Shape": "Circle"})
//	ref := map[string]interface{}{"Circle": &Circle{}, "Geo": &Geo{}}
//
//	var geo Geo
//	err := UnmarshalSpec(hcl, &geo, spec, ref)
//
// Returns an error if decoding fails or if referenced types are not in ref map.
func UnmarshalSpec(hclData []byte, current interface{}, spec *utils.Struct, ref map[string]interface{}, labels ...string) error {
	node, ref := utils.NewTreeCtyFunction(ref)
	return UnmarshalSpecTree(node, hclData, current, spec, ref, labels...)
}

// UnmarshalSpecTree decodes HCL data with interface specifications at a specific tree node.
//
// This function extends UnmarshalSpec by operating within a specific node of the tree
// structure used for managing HCL variable scope and function context. Use this when
// you need fine-grained control over the variable resolution context.
//
// Parameters:
//   - node: tree node for variable scope management (created with utils.NewTreeCtyFunction)
//   - hclData: HCL data as bytes
//   - current: pointer to target struct, map[string]interface{}, or []interface{}
//   - spec: struct specification describing interface field types (nil for no interfaces)
//   - ref: type registry mapping type names to zero-value instances
//   - labels: optional HCL label values for labeled blocks
//
// The tree node provides context for evaluating HCL expressions that reference variables
// or call functions. Each node in the tree represents a scope level in the HCL configuration.
//
// Example:
//
//	node, ref := utils.NewTreeCtyFunction(ref)
//	spec, _ := utils.NewStruct("Config", map[string]interface{}{
//	    "Database": "PostgresDB",
//	})
//
//	var cfg Config
//	err := UnmarshalSpecTree(node, hclBytes, &cfg, spec, ref)
//
// Returns an error if decoding fails, if the target is not a pointer, or if referenced
// types are not found in the ref map.
func UnmarshalSpecTree(node *utils.Tree, hclData []byte, current interface{}, spec *utils.Struct, ref map[string]interface{}, labels ...string) error {
	reflectValue := reflect.ValueOf(current)
	if reflectValue.Kind() != reflect.Pointer {
		return fmt.Errorf("non-pointer or nil data")
	}
	reflectValue = reflectValue.Elem()

	// Handle map[string]interface{} and []interface{} types
	switch reflectValue.Kind() {
	case reflect.Map:
		return unmarshalToMap(node, hclData, current)
	case reflect.Slice:
		return unmarshalToSlice(node, hclData, current)
	default:
	}

	// Validate struct type
	structType := reflect.TypeOf(current)
	if structType.Kind() != reflect.Pointer {
		return fmt.Errorf("non-pointer or nil data")
	}
	structType = structType.Elem()
	if structType.Kind() == reflect.Pointer {
		structType = structType.Elem()
	}
	if structType.Kind() != reflect.Struct {
		return fmt.Errorf("non-struct object")
	}

	// Get spec fields or create empty map
	var objectMap map[string]*utils.Value
	if spec != nil {
		objectMap = spec.GetFields()
	}
	if objectMap == nil {
		objectMap = make(map[string]*utils.Value)
	}

	// Parse HCL file
	file, hclBody, err := parseHCLFile(hclData)
	if err != nil {
		return err
	}

	// Evaluate expressions and find null attributes
	nullAttrs, err := evaluateExpressions(ref, node, file, hclBody)
	if err != nil {
		return err
	}

	// Register blocks in tree
	addBlocksToTree(node, hclBody.Blocks)

	// Categorize struct fields
	fieldCategories, err := categorizeStructFields(structType, objectMap, ref, nullAttrs)
	if err != nil {
		return err
	}

	// Parse and separate HCL body into labels, attributes, and blocks
	parseResult, err := categorizeHCLBody(node, file, hclBody, nullAttrs, fieldCategories.BlockFields, fieldCategories.InterfaceFields, fieldCategories.SimpleFields, fieldCategories.Labels)
	if err != nil {
		return err
	}

	// Create a copy of the target struct to populate
	targetValue := reflect.ValueOf(&current).Elem()
	updatedValue := reflect.New(targetValue.Elem().Type()).Elem()
	updatedValue.Set(targetValue.Elem())

	// Process label fields
	if err := processLabels(fieldCategories.Labels, updatedValue, parseResult.LabelExprs, labels); err != nil {
		return err
	}

	// Process simple fields (strings, numbers, etc.)
	processSimpleFields(fieldCategories.SimpleFields, parseResult.SimpleFieldsValue, updatedValue, parseResult.ExistingAttrs)

	// Process map/slice interface fields
	if err := processMapOrSliceFields(ref, node, file, fieldCategories.InterfaceFields, parseResult.InterfaceAttrs, parseResult.InterfaceBlocks, updatedValue); err != nil {
		return err
	}

	// Process complex block fields (Map2Struct, MapStruct, ListStruct, SingleStruct)
	if err := processBlockFields(node, file, ref, fieldCategories.BlockFields, parseResult.BlockData, objectMap, updatedValue); err != nil {
		return err
	}

	// Apply all changes to the original struct
	targetValue.Set(updatedValue)

	return nil
}

// tryUnmarshalWithCustom attempts to unmarshal using custom Unmarshaler interface first,
// falling back to standard UnmarshalSpecTree if the interface is not implemented.
//
// This function provides a two-tier unmarshaling strategy:
//  1. Check if the type implements Unmarshaler interface (custom unmarshaling)
//  2. If not, use standard UnmarshalSpecTree with spec and ref
//
// Parameters:
//   - subnode: tree node for variable scope context
//   - hclData: HCL bytes to unmarshal
//   - trial: target object instance
//   - nextStruct: spec describing interface field types
//   - ref: type registry for resolving type names
//   - labels: optional HCL label values
//
// Returns error if unmarshaling fails.
func tryUnmarshalWithCustom(subnode *utils.Tree, hclData []byte, trial interface{}, nextStruct *utils.Struct, ref map[string]interface{}, labels ...string) error {
	unmarshaler, ok := trial.(Unmarshaler)
	if ok {
		return unmarshaler.UnmarshalHCL(hclData, labels...)
	}
	return UnmarshalSpecTree(subnode, hclData, trial, nextStruct, ref, labels...)
}

// hclBodyParseResult holds the categorized results from parsing an HCL body
type hclBodyParseResult struct {
	LabelExprs      map[string]hclsyntax.Expression   // Label field expressions
	ExistingAttrs   map[string]bool                   // Simple attributes that exist
	SimpleFieldsValue reflect.Value                   // Decoded simple fields struct
	InterfaceAttrs  map[string]*hclsyntax.Attribute   // Dynamic interface attributes
	InterfaceBlocks map[string][]*hclsyntax.Block     // Dynamic interface blocks
	BlockData       map[string][]*hclsyntax.Block     // Complex block data
}

// categorizeHCLBody parses and categorizes HCL body elements into different field types
func categorizeHCLBody(node *utils.Tree, file *hcl.File, hclBody *hclsyntax.Body, nullAttrs []string, blockFields, interfaceFields, newFields, newLabels []reflect.StructField) (*hclBodyParseResult, error) {
	result := &hclBodyParseResult{
		BlockData:       make(map[string][]*hclsyntax.Block),
		InterfaceBlocks: make(map[string][]*hclsyntax.Block),
	}

	body := &hclsyntax.Body{SrcRange: hclBody.SrcRange, EndRange: hclBody.EndRange}

	blockTags := buildTagIndex(blockFields)
	interfaceTags := buildTagIndex(interfaceFields)
	labelTags := buildTagIndex(newLabels)
	for attrName, attr := range hclBody.Attributes {
		if slices.Contains(nullAttrs, attrName) {
			continue
		}
		if interfaceTags[attrName] {
			if result.InterfaceAttrs == nil {
				result.InterfaceAttrs = make(map[string]*hclsyntax.Attribute)
			}
			result.InterfaceAttrs[attrName] = attr
		} else if labelTags[attrName] {
			if result.LabelExprs == nil {
				result.LabelExprs = make(map[string]hclsyntax.Expression)
			}
			result.LabelExprs[attrName] = attr.Expr
		} else if blockTags[attrName] { // this MUST BE hash or slice with equal sign.
			// Unmarshal []any produces an equal sign (unmarshal a map[string]any does not)
			// Equal sign results in suxh N attribute. It is recorded in oriref and there is a struct associated.
			if literalExpr, ok := attr.Expr.(*hclsyntax.LiteralValueExpr); !ok || !literalExpr.Val.CanIterateElements() {
				return nil, fmt.Errorf("unknown expression type %T", literalExpr)
			}
			start := attr.EqualsRange.End.Byte + 1
			attributeContent := string(file.Bytes[start:attr.SrcRange.End.Byte])
			blockPattern := regexp.MustCompile(`(?s){[^}]+}`)
			// the starting and ending positions of each matched string block, including the braces
			indices := blockPattern.FindAllStringIndex(attributeContent, -1)
			// there is only one block in case of hash; there would be multiple blocks in case of slice
			for _, item := range indices {
				block := &hclsyntax.Block{
					Type: attrName,
					OpenBraceRange: hcl.Range{
						End: hcl.Pos{Byte: start + item[0] + 1}, // remove leading brace
					},
					CloseBraceRange: hcl.Range{
						Start: hcl.Pos{Byte: start + item[1] - 1}, // remove trailing brace
					},
				}
				result.BlockData[attrName] = append(result.BlockData[attrName], block)
			}
			node.AddNode(attrName)
		} else {
			if body.Attributes == nil {
				body.Attributes = make(map[string]*hclsyntax.Attribute)
			}
			body.Attributes[attrName] = attr
			if result.ExistingAttrs == nil {
				result.ExistingAttrs = make(map[string]bool)
			}
			result.ExistingAttrs[attrName] = true
		}
	}

	for _, block := range hclBody.Blocks {
		tag := block.Type
		if blockTags[tag] {
			result.BlockData[tag] = append(result.BlockData[tag], block)
		} else if interfaceTags[tag] {
			result.InterfaceBlocks[tag] = append(result.InterfaceBlocks[tag], block)
		} else {
			body.Blocks = append(body.Blocks, block)
		}
	}

	newType := reflect.StructOf(newFields)
	rawValue := reflect.New(newType).Elem()

	// Convert each simple field from its cty.Value (already evaluated and stored in node)
	// to the exact field type using gocty.FromCtyValue
	for i, field := range newFields {
		tag := (parseHCLTag(field.Tag))[0]

		// Get the already-evaluated cty.Value from the tree node
		ctyValInterface, ok := node.Data.Load(tag)
		if !ok {
			continue // Field not present in HCL, keep zero value
		}

		ctyVal, ok := ctyValInterface.(cty.Value)
		if !ok {
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Internal error",
				Detail:   fmt.Sprintf("Field %s: expected cty.Value, got %T", tag, ctyValInterface),
			}}
		}

		// Convert to the exact field type
		nativeVal, err := utils.ConvertCtyToFieldType(ctyVal, field.Type)
		if err != nil {
			return nil, hcl.Diagnostics{{
				Severity: hcl.DiagError,
				Summary:  "Type conversion error",
				Detail:   fmt.Sprintf("Field %s: %v", tag, err),
			}}
		}

		rawValue.Field(i).Set(reflect.ValueOf(nativeVal))
	}

	result.SimpleFieldsValue = rawValue
	return result, nil
}

// getBlockBytes extracts the content bytes and labels from an HCL block.
// Returns the block body (content between braces) and the block's labels.
//
// For example, given: service "api" "web" { port = 8080 }
// Returns: ([]byte("port = 8080"), []string{"api", "web"}, nil)
//
// Parameters:
//   - block: HCL syntax block to extract from
//   - file: parsed HCL file containing the source bytes
//
// Returns block content bytes, labels, and error if block is nil.
func getBlockBytes(block *hclsyntax.Block, file *hcl.File) ([]byte, []string, error) {
	if block == nil {
		return nil, nil, fmt.Errorf("block not found")
	}
	openRange := block.OpenBraceRange
	closeRange := block.CloseBraceRange
	hclBytes := file.Bytes[openRange.End.Byte:closeRange.Start.Byte]
	return hclBytes, block.Labels, nil
}

// buildTagIndex creates a lookup map of HCL tag names for quick field categorization.
// Used during HCL body parsing to efficiently check if an attribute/block belongs to a specific field category.
//
// For example, given fields with tags "name", "port", "service":
// Returns: map[string]bool{"name": true, "port": true, "service": true}
//
// This enables O(1) lookup when categorizing HCL attributes and blocks into:
//   - Simple fields (decoded by gohcl)
//   - Block fields (complex nested structures)
//   - Interface fields (dynamic map/slice types)
//   - Label fields (block label values)
//
// Parameters:
//   - fields: slice of struct fields to index
//
// Returns a map with HCL tag names as keys, all values set to true.
func buildTagIndex(fields []reflect.StructField) map[string]bool {
	tagIndex := make(map[string]bool)
	for _, field := range fields {
		tag := parseHCLTag(field.Tag)[0]
		tagIndex[tag] = true
	}
	return tagIndex
}

// structFieldCategories holds categorized struct fields for unmarshaling
type structFieldCategories struct {
	Labels         []reflect.StructField // Fields marked with "label" modifier
	SimpleFields   []reflect.StructField // Normal fields decoded with gohcl
	BlockFields    []reflect.StructField // Block fields decoded individually
	InterfaceFields []reflect.StructField // Dynamic map[string]interface{} or []interface{}
}

// categorizeStructFields analyzes struct fields and categorizes them into different types
func categorizeStructFields(structType reflect.Type, objectMap map[string]*utils.Value, ref map[string]interface{}, nullAttrs []string) (*structFieldCategories, error) {
	categories := &structFieldCategories{}
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		name := field.Name
		if !unicode.IsUpper([]rune(name)[0]) {
			continue
		}
		tag := parseHCLTag(field.Tag)[0]
		if slices.Contains(nullAttrs, tag) {
			continue
		}
		tagModifier := parseHCLTag(field.Tag)[1]
		if strings.ToLower(tagModifier) == TagModifierLabel {
			categories.Labels = append(categories.Labels, field)
			continue
		}
		if tag == TagIgnore || (len(tag) >= 2 && tag[len(tag)-2:] == TagIgnoreSuffix) {
			continue
		}
		if _, ok := objectMap[name]; ok {
			categories.BlockFields = append(categories.BlockFields, field)
			continue
		}

		if tag == "" {
			switch fieldType.Kind() {
			case reflect.Struct:
				nested, err := categorizeStructFields(fieldType, objectMap, ref, nullAttrs)
				if err != nil {
					return nil, err
				}
				categories.Labels = append(categories.Labels, nested.Labels...)
				categories.SimpleFields = append(categories.SimpleFields, nested.SimpleFields...)
				categories.BlockFields = append(categories.BlockFields, nested.BlockFields...)
				categories.InterfaceFields = append(categories.InterfaceFields, nested.InterfaceFields...)
			default:
			}
			continue
		}

		if fieldType.Kind() == reflect.Struct {
			typeName := fieldType.String()
			ref[typeName] = reflect.New(fieldType).Interface()
			valueSpec, err := utils.NewValue(typeName)
			if err != nil {
				return nil, err
			}
			objectMap[field.Name] = valueSpec
			categories.BlockFields = append(categories.BlockFields, field)
		} else if fieldType.Kind() == reflect.Map && fieldType.Key().Kind() == reflect.Array && fieldType.Key().Len() == 2 {
			// this is map[[2]string]string
			elemType := fieldType.Elem()
			typeName := elemType.String()

			switch elemType.Kind() {
			case reflect.Struct:
				ref[typeName] = reflect.New(elemType).Interface()
			case reflect.Pointer:
				ref[typeName] = reflect.New(elemType.Elem()).Interface()
			case reflect.Interface:
				categories.InterfaceFields = append(categories.InterfaceFields, field)
				continue
			default:
				categories.SimpleFields = append(categories.SimpleFields, field)
				continue
			}
			// use 2 empty strings here as key, then firstFirst in unmarshaling as default
			valueSpec, err := utils.NewValue(map[[2]string]string{{"", ""}: typeName})
			if err != nil {
				return nil, err
			}
			objectMap[field.Name] = valueSpec
			categories.BlockFields = append(categories.BlockFields, field)
		} else if fieldType.Kind() == reflect.Slice || fieldType.Kind() == reflect.Map {
			elemType := fieldType.Elem()
			typeName := elemType.String()

			switch elemType.Kind() {
			case reflect.Struct:
				ref[typeName] = reflect.New(elemType).Interface()
			case reflect.Pointer:
				ref[typeName] = reflect.New(elemType.Elem()).Interface()
			case reflect.Interface:
				categories.InterfaceFields = append(categories.InterfaceFields, field)
				continue
			default:
				categories.SimpleFields = append(categories.SimpleFields, field)
				continue
			}

			valueSpec, err := utils.NewValue([]string{typeName})
			if err != nil {
				return nil, err
			}
			objectMap[field.Name] = valueSpec
			categories.BlockFields = append(categories.BlockFields, field)
		} else {
			categories.SimpleFields = append(categories.SimpleFields, field)
		}
	}
	return categories, nil
}

// getLabels extracts label field values from a struct instance.
// Scans struct fields for those tagged with "label" modifier and returns their string values.
//
// HCL blocks can have up to 2 labels:
//   - Single label: service "api" { ... }
//   - Two labels: service "http" "api" { ... }
//
// For example, given a struct:
//
//	type Service struct {
//	    Type string `hcl:"type,label"`
//	    Name string `hcl:"name,label"`
//	    Port int    `hcl:"port"`
//	}
//
// With values Type="http", Name="api", returns: ("http", "api")
//
// Parameters:
//   - current: pointer to struct instance to extract labels from
//
// Returns up to two label values as strings. Returns empty strings if no labels found.
func getLabels(current interface{}) (string, string) {
	structType := reflect.TypeOf(current).Elem()
	numFields := structType.NumField()
	structValue := reflect.ValueOf(current).Elem()

	labelCount := 0
	var key0, key1 string
	for i := 0; i < numFields; i++ {
		field := structType.Field(i)
		fieldValue := structValue.Field(i)
		tagParts := parseHCLTag(field.Tag)
		if strings.ToLower(tagParts[1]) == TagModifierLabel {
			if labelCount == 0 {
				key0 = fieldValue.String()
			} else {
				key1 = fieldValue.String()
			}
			labelCount++
		}
	}
	return key0, key1
}

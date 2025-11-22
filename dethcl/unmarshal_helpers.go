package dethcl

import (
	"fmt"
	"reflect"

	"github.com/genelet/horizon/utils"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// unmarshalToMap handles unmarshaling HCL data to a map[string]interface{}.
// Used when the target type is a dynamic map without a defined struct schema.
//
// The function decodes HCL into a nested map structure where:
//   - Attributes become map entries with their values
//   - Blocks without labels become nested maps
//   - Blocks with labels become map[label]value
//
// Parameters:
//   - node: tree node for variable scope
//   - dat: HCL data bytes
//   - current: pointer to map[string]interface{} to populate
//
// Returns error if parsing or decoding fails.
func unmarshalToMap(node *utils.Tree, dat []byte, current interface{}) error {
	obj, err := decodeMap(nil, node, dat)
	if err != nil {
		return err
	}
	x := current.(*map[string]interface{})
	for k, v := range obj {
		(*x)[k] = v
	}
	return nil
}

// unmarshalToSlice handles unmarshaling HCL data to a []interface{}.
// Used when the target type is a dynamic slice without a defined element schema.
//
// The function decodes HCL array syntax into a slice where each element
// can be a primitive value, nested map, or nested slice.
//
// For example, HCL: ["str", 123, {key = "val"}, [1, 2]]
// Becomes: []interface{}{"str", 123, map[string]interface{}{"key": "val"}, []interface{}{1, 2}}
//
// Parameters:
//   - node: tree node for variable scope
//   - dat: HCL data bytes (should be array syntax)
//   - current: pointer to []interface{} to append to
//
// Returns error if parsing or decoding fails.
func unmarshalToSlice(node *utils.Tree, dat []byte, current interface{}) error {
	obj, err := decodeSlice(nil, node, dat)
	if err != nil {
		return err
	}
	x := current.(*[]interface{})
	*x = append(*x, obj...)
	return nil
}

// parseHCLFile parses raw HCL bytes into an AST (abstract syntax tree).
// This is the first step in unmarshaling, converting HCL text into structured data.
//
// The function:
//   - Parses HCL syntax using hashicorp/hcl parser
//   - Validates syntax and reports errors
//   - Extracts the body containing attributes and blocks
//
// Parameters:
//   - dat: HCL configuration bytes
//
// Returns parsed file, body, and any parsing errors.
func parseHCLFile(dat []byte) (*hcl.File, *hclsyntax.Body, error) {
	file, diags := hclsyntax.ParseConfig(dat, generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, nil, fmt.Errorf("failed to parse HCL: %w", diags)
	}
	bd := file.Body.(*hclsyntax.Body)
	return file, bd, nil
}

// evaluateExpressions evaluates HCL expressions and converts them to concrete values.
// This handles HCL's expression evaluation including variables, functions, and references.
//
// The function:
//   - Evaluates each attribute expression using the tree context (variables/functions)
//   - Converts expressions to cty values (HCL's type system)
//   - Replaces expressions with their evaluated values
//   - Tracks null values to skip them during unmarshaling
//   - Updates the tree with evaluated attribute values for reference by nested blocks
//
// For example, given HCL with variable references:
//   name = var.service_name
//   port = local.default_port
//
// The function resolves these references using the tree context.
//
// Parameters:
//   - ref: type registry for resolving struct types
//   - node: tree node containing variable scope and functions
//   - file: parsed HCL file
//   - bd: HCL body with attributes to evaluate
//
// Returns list of attribute names with null values (to be ignored), or error if evaluation fails.
func evaluateExpressions(ref map[string]interface{}, node *utils.Tree, file *hcl.File, bd *hclsyntax.Body) ([]string, error) {
	var kNulls []string
	for k, v := range bd.Attributes {
		cv, err := utils.ExpressionToCty(ref, node, v.Expr)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate expression for %q: %w", k, err)
		}
		if cv.IsNull() {
			kNulls = append(kNulls, k)
		}
		v.Expr = utils.CtyToExpression(cv, v.Range())
		node.AddItem(k, cv)
	}
	return kNulls, nil
}

// addBlocksToTree registers block types and labels in the Tree structure.
func addBlocksToTree(node *utils.Tree, blocks []*hclsyntax.Block) {
	for _, block := range blocks {
		node.AddNodes(block.Type, block.Labels...)
	}
}

// processLabels fills in label field values from parsed HCL or provided labels.
// Labels can come from two sources:
// 1. Parsed from HCL (in labelExprs)
// 2. Passed from parent unmarshal (in labels parameter)
func processLabels(labelFields []reflect.StructField, oriTobe reflect.Value, labelExprs map[string]hclsyntax.Expression, labels []string) error {
	if labelExprs != nil {
		for _, field := range labelFields {
			name := field.Name
			f := oriTobe.Elem().FieldByName(name)
			tag := (parseHCLTag(field.Tag))[0]
			expr, ok := labelExprs[tag]
			if ok {
				cv, diags := expr.Value(nil)
				if diags.HasErrors() {
					return fmt.Errorf("failed to evaluate label %q: %w", tag, diags)
				}
				label := cv.AsString()
				f.Set(reflect.ValueOf(label))
			}
		}
	}

	// Add missing labels from parent context
	if labels != nil && labelFields != nil && len(labels) == len(labelFields) {
		for i, field := range labelFields {
			name := field.Name
			f := oriTobe.Elem().FieldByName(name)
			if f.String() == "" {
				label := labels[i]
				f.Set(reflect.ValueOf(label))
			}
		}
	}
	return nil
}

// processSimpleFields copies simple field values from the decoded struct to the target.
func processSimpleFields(newFields []reflect.StructField, rawValue reflect.Value, oriTobe reflect.Value, existingAttrs map[string]bool) {
	for i, field := range newFields {
		name := field.Name
		tag := (parseHCLTag(field.Tag))[0]
		if _, ok := existingAttrs[tag]; ok {
			rawField := rawValue.Field(i)
			f := oriTobe.Elem().FieldByName(name)
			f.Set(rawField)
		}
	}
}

// processMapOrSliceFields handles dynamic interface fields (map[string]interface{} and []interface{}).
// These fields can contain any HCL structure and are decoded into generic Go types.
//
// The function:
//   - Locates the field's HCL source (either attribute or block)
//   - Extracts the raw HCL bytes for that field
//   - Decodes into map[string]interface{} or []interface{} based on field type
//   - Sets the decoded value on the target struct
//
// This enables flexible schemas where field contents are not known at compile time.
//
// For example, given:
//
//	type Config struct {
//	    Settings map[string]interface{} `hcl:"settings"`
//	}
//
// The HCL: settings = { key1 = "val", nested = { key2 = 123 } }
// Becomes: map[string]interface{}{"key1": "val", "nested": map[string]interface{}{"key2": 123}}
//
// Parameters:
//   - ref: type registry
//   - node: tree node for variable scope
//   - file: parsed HCL file with source bytes
//   - decFields: interface fields to process
//   - decattrs: map of attribute names to their HCL syntax
//   - decblock: map of block names to their HCL syntax
//   - oriTobe: target struct value to populate
//
// Returns error if field decoding fails.
func processMapOrSliceFields(ref map[string]interface{}, node *utils.Tree, file *hcl.File, decFields []reflect.StructField, decattrs map[string]*hclsyntax.Attribute, decblock map[string][]*hclsyntax.Block, oriTobe reflect.Value) error {
	for _, field := range decFields {
		var bs []byte
		var err error
		tag := (parseHCLTag(field.Tag))[0]
		if attr, ok := decattrs[tag]; ok {
			bs = file.Bytes[attr.EqualsRange.End.Byte:attr.SrcRange.End.Byte]
		} else if blkd, ok := decblock[tag]; ok {
			bs, _, err = getBlockBytes(blkd[0], file)
			if err != nil {
				return err
			}
		} else {
			continue
		}

		name := field.Name
		typ := field.Type
		f := oriTobe.Elem().FieldByName(name)
		if typ.Kind() == reflect.Slice {
			obj, err := decodeSlice(ref, node, bs)
			if err != nil {
				return fmt.Errorf("field %s: failed to decode slice: %w", name, err)
			}
			f.Set(reflect.ValueOf(obj))
		} else {
			obj, err := decodeMap(ref, node, bs)
			if err != nil {
				return fmt.Errorf("field %s: failed to decode map: %w", name, err)
			}
			f.Set(reflect.ValueOf(obj))
		}
	}
	return nil
}

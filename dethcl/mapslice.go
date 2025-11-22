package dethcl

import (
	"fmt"
	"strings"

	"github.com/genelet/horizon/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func decodeSlice(ref map[string]interface{}, node *utils.Tree, hclBytes []byte) ([]interface{}, error) {
	file, diags := hclsyntax.ParseConfig(append([]byte(TempAttributeName+" = "), hclBytes...), generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse slice HCL: %w", diags.Errs()[0])
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("expected hclsyntax.Body, got %T", file.Body)
	}

	attr, exists := body.Attributes[TempAttributeName]
	if !exists {
		return nil, fmt.Errorf("temporary attribute %q not found in parsed HCL", TempAttributeName)
	}

	tuple, ok := attr.Expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil, fmt.Errorf("expected array/tuple expression, got %T", attr.Expr)
	}

	var object []interface{}
	for index, item := range tuple.Exprs {
		value, err := expressionToNative(ref, node, file, index, item)
		if err != nil {
			return nil, err
		}
		object = append(object, value)
	}
	return object, nil
}

func decodeMap(ref map[string]interface{}, node *utils.Tree, hclBytes []byte) (map[string]interface{}, error) {
	trimmed := strings.TrimSpace(string(hclBytes))
	if trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		return decodeObjectConsExpr(ref, node, hclBytes)
	}
	file, diags := hclsyntax.ParseConfig(hclBytes, generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse map HCL: %w", diags.Errs()[0])
	}

	return decodeBody(ref, node, file, file.Body.(*hclsyntax.Body))
}

func decodeBody(ref map[string]interface{}, node *utils.Tree, file *hcl.File, body *hclsyntax.Body) (map[string]interface{}, error) {
	object := make(map[string]interface{})
	for key, item := range body.Attributes {
		value, err := expressionToNative(ref, node, file, key, item.Expr, item)
		if err != nil {
			return nil, err
		}
		object[key] = value
	}

	var sliceBodies map[string][]*hclsyntax.Body
	counts := make(map[string]int)
	var mapBodies map[string]map[string]*hclsyntax.Body
	var map2Bodies map[string]map[string]map[string]*hclsyntax.Body
	for _, item := range body.Blocks {
		switch len(item.Labels) {
		case 0:
			if sliceBodies == nil {
				sliceBodies = make(map[string][]*hclsyntax.Body)
			}
			sliceBodies[item.Type] = append(sliceBodies[item.Type], item.Body)
			counts[item.Type]++
		case 1:
			if mapBodies == nil {
				mapBodies = make(map[string]map[string]*hclsyntax.Body)
			}
			if mapBodies[item.Type] == nil {
				mapBodies[item.Type] = make(map[string]*hclsyntax.Body)
			}
			mapBodies[item.Type][item.Labels[0]] = item.Body
		case 2:
			if map2Bodies == nil {
				map2Bodies = make(map[string]map[string]map[string]*hclsyntax.Body)
			}
			if map2Bodies[item.Type] == nil {
				map2Bodies[item.Type] = make(map[string]map[string]*hclsyntax.Body)
			}
			if map2Bodies[item.Type][item.Labels[0]] == nil {
				map2Bodies[item.Type][item.Labels[0]] = make(map[string]*hclsyntax.Body)
			}
			map2Bodies[item.Type][item.Labels[0]][item.Labels[1]] = item.Body
		default:
			return nil, fmt.Errorf("unsupported number of labels (%d) for block type %q: expected 0, 1, or 2", len(item.Labels), item.Type)
		}
	}

	for key, bodies := range sliceBodies {
		var values []interface{}
		keyNode := node.AddNode(key)
		for i, body := range bodies {
			subNode := keyNode.AddNode(fmt.Sprintf("%d", i))
			decoded, err := decodeBody(ref, subNode, file, body)
			if err != nil {
				return nil, err
			}
			values = append(values, decoded)
		}
		if counts[key] > 1 {
			object[key] = values
		} else {
			object[key] = values[0]
		}
	}

	for key, bodies := range mapBodies {
		valueMap := make(map[string]interface{})
		keyNode := node.AddNode(key)
		for mapKey, body := range bodies {
			subNode := keyNode.AddNode(mapKey)
			decoded, err := decodeBody(ref, subNode, file, body)
			if err != nil {
				return nil, err
			}
			valueMap[mapKey] = decoded
		}
		object[key] = valueMap
	}

	for key, bodies := range map2Bodies {
		outerMap := make(map[string]interface{})
		keyNode := node.AddNode(key)
		for firstKey, bodies2 := range bodies {
			innerMap := make(map[string]interface{})
			keyNode2 := keyNode.AddNode(firstKey)
			for secondKey, body := range bodies2 {
				subNode := keyNode2.AddNode(secondKey)
				decoded, err := decodeBody(ref, subNode, file, body)
				if err != nil {
					return nil, err
				}
				innerMap[secondKey] = decoded
			}
			outerMap[firstKey] = innerMap
		}
		object[key] = outerMap
	}

	return object, nil
}

func decodeObjectConsExpr(ref map[string]interface{}, node *utils.Tree, hclBytes []byte) (map[string]interface{}, error) {
	file, diags := hclsyntax.ParseConfig(append([]byte(TempAttributeName+" = "), hclBytes...), generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse object expression: %w", diags.Errs()[0])
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("expected hclsyntax.Body, got %T", file.Body)
	}

	attr, exists := body.Attributes[TempAttributeName]
	if !exists {
		return nil, fmt.Errorf("temporary attribute %q not found in parsed HCL", TempAttributeName)
	}

	exprs, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("expected object/map expression, got %T", attr.Expr)
	}

	object := make(map[string]interface{})
	for _, item := range exprs.Items {
		keyExpr, ok := item.KeyExpr.(*hclsyntax.ObjectConsKeyExpr)
		if !ok {
			return nil, fmt.Errorf("expected ObjectConsKeyExpr for map key, got %T", item.KeyExpr)
		}

		key, diags := keyExpr.Value(nil)
		if diags.HasErrors() {
			return nil, (diags.Errs())[0]
		}

		value, err := expressionToNative(ref, node, file, key.AsString(), item.ValueExpr)
		if err != nil {
			return nil, err
		}
		object[key.AsString()] = value
	}
	return object, nil
}

func expressionToNative(ref map[string]interface{}, node *utils.Tree, file *hcl.File, key interface{}, item hclsyntax.Expression, attr ...*hclsyntax.Attribute) (interface{}, error) {
	switch exprType := item.(type) {
	case *hclsyntax.TupleConsExpr: // array
		sourceRange := exprType.SrcRange
		hclBytes := file.Bytes[sourceRange.Start.Byte:sourceRange.End.Byte]
		subNode := node.AddNode(fmt.Sprintf("%v", key))
		return decodeSlice(ref, subNode, hclBytes)
	case *hclsyntax.ObjectConsExpr: // map
		sourceRange := exprType.SrcRange
		hclBytes := file.Bytes[sourceRange.Start.Byte:sourceRange.End.Byte]
		subNode := node.AddNode(fmt.Sprintf("%v", key))
		return decodeMap(ref, subNode, hclBytes)
	case *hclsyntax.FunctionCallExpr:
		if exprType.Name == "null" {
			return nil, nil
		}
	default:
	}

	ctyValue, err := utils.ExpressionToCty(ref, node, item)
	if err != nil {
		return nil, err
	}

	if attr != nil {
		attr[0].Expr = utils.CtyToExpression(ctyValue, attr[0].Expr.Range())
	}
	//item = utils.CtyToExpression(ctyValue, item.Range())

	node.AddItem(fmt.Sprintf("%v", key), ctyValue)

	return utils.CtyToNative(ctyValue)
}

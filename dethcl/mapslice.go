package dethcl

import (
	"fmt"
	"strings"

	"github.com/genelet/horizon/utils"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func decodeSlice(ref map[string]any, node *utils.Tree, hclBytes []byte) ([]any, error) {
	file, diags := hclsyntax.ParseConfig(append([]byte(tempAttributeName+" = "), hclBytes...), generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse slice HCL: %w", diags.Errs()[0])
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("expected hclsyntax.Body, got %T", file.Body)
	}

	attr, exists := body.Attributes[tempAttributeName]
	if !exists {
		return nil, fmt.Errorf("temporary attribute %q not found in parsed HCL", tempAttributeName)
	}

	tuple, ok := attr.Expr.(*hclsyntax.TupleConsExpr)
	if !ok {
		return nil, fmt.Errorf("expected array/tuple expression, got %T", attr.Expr)
	}

	return decodeTuple(ref, node, file, tuple)
}

func decodeMap(ref map[string]any, node *utils.Tree, hclBytes []byte) (map[string]any, error) {
	trimmed := strings.TrimSpace(string(hclBytes))
	if len(trimmed) == 0 {
		return make(map[string]any), nil
	}
	if trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		return decodeObjectConsExpr(ref, node, hclBytes)
	}
	file, diags := hclsyntax.ParseConfig(hclBytes, generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse map HCL: %w", diags.Errs()[0])
	}

	return decodeBody(ref, node, file, file.Body.(*hclsyntax.Body))
}

func decodeBody(ref map[string]any, node *utils.Tree, file *hcl.File, body *hclsyntax.Body) (map[string]any, error) {
	object := make(map[string]any)
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
		var values []any
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
		valueMap := make(map[string]any)
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
		outerMap := make(map[string]any)
		keyNode := node.AddNode(key)
		for firstKey, bodies2 := range bodies {
			innerMap := make(map[string]any)
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

func decodeObjectConsExpr(ref map[string]any, node *utils.Tree, hclBytes []byte) (map[string]any, error) {
	file, diags := hclsyntax.ParseConfig(append([]byte(tempAttributeName+" = "), hclBytes...), generateTempHCLFileName(), hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, fmt.Errorf("failed to parse object expression: %w", diags.Errs()[0])
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("expected hclsyntax.Body, got %T", file.Body)
	}

	attr, exists := body.Attributes[tempAttributeName]
	if !exists {
		return nil, fmt.Errorf("temporary attribute %q not found in parsed HCL", tempAttributeName)
	}

	exprs, ok := attr.Expr.(*hclsyntax.ObjectConsExpr)
	if !ok {
		return nil, fmt.Errorf("expected object/map expression, got %T", attr.Expr)
	}

	return decodeObject(ref, node, file, exprs)
}

func decodeTuple(ref map[string]any, node *utils.Tree, file *hcl.File, tuple *hclsyntax.TupleConsExpr) ([]any, error) {
	var object []any
	for index, item := range tuple.Exprs {
		value, err := expressionToNative(ref, node, file, index, item)
		if err != nil {
			return nil, err
		}
		object = append(object, value)
	}
	return object, nil
}

func decodeObject(ref map[string]any, node *utils.Tree, file *hcl.File, exprs *hclsyntax.ObjectConsExpr) (map[string]any, error) {
	object := make(map[string]any)
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

func expressionToNative(ref map[string]any, node *utils.Tree, file *hcl.File, key any, item hclsyntax.Expression, attr ...*hclsyntax.Attribute) (any, error) {
	switch exprType := item.(type) {
	case *hclsyntax.TupleConsExpr: // array
		subNode := node.AddNode(fmt.Sprintf("%v", key))
		return decodeTuple(ref, subNode, file, exprType)
	case *hclsyntax.ObjectConsExpr: // map
		subNode := node.AddNode(fmt.Sprintf("%v", key))
		return decodeObject(ref, subNode, file, exprType)
	default:
	}

	ctyValue, err := utils.ExpressionToCty(ref, node, item)
	if err != nil {
		return nil, err
	}

	if attr != nil {
		attr[0].Expr = utils.CtyToExpression(ctyValue, attr[0].Expr.Range())
	}

	node.AddItem(fmt.Sprintf("%v", key), ctyValue)

	return utils.CtyToNative(ctyValue)
}

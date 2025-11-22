package utils

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/convert"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/gocty"

	ilang "github.com/genelet/horizon/internal/lang"
)

// CtyToExpression converts a cty.Value back into an HCL expression.
// This is useful for modifying HCL expressions during parsing and evaluation.
//
// Supports conversion of:
//   - Primitive types (string, number, bool)
//   - Lists/tuples of primitives
//   - Maps/objects of primitives
//
// For complex nested types, returns a LiteralValueExpr with the cty.Value.
func CtyToExpression(cv cty.Value, rng hcl.Range) hclsyntax.Expression {
	switch cv.Type() {
	case cty.String, cty.Number, cty.Bool:
		return &hclsyntax.LiteralValueExpr{Val: cv, SrcRange: rng}
	case cty.List(cty.String), cty.List(cty.Number), cty.List(cty.Bool):
		var exprs []hclsyntax.Expression
		for _, item := range cv.AsValueSlice() {
			exprs = append(exprs, &hclsyntax.LiteralValueExpr{Val: item, SrcRange: rng})
		}
		return &hclsyntax.TupleConsExpr{Exprs: exprs, SrcRange: rng}
	case cty.Map(cty.String), cty.Map(cty.Number), cty.Map(cty.Bool):
		var items []hclsyntax.ObjectConsItem
		for k, item := range cv.AsValueMap() {
			items = append(items, hclsyntax.ObjectConsItem{
				KeyExpr:   &hclsyntax.LiteralValueExpr{Val: cty.StringVal(k), SrcRange: rng},
				ValueExpr: &hclsyntax.LiteralValueExpr{Val: item, SrcRange: rng},
			})
		}
		return &hclsyntax.ObjectConsExpr{Items: items, SrcRange: rng}
	default:
	}
	// just use the default seems to be ok
	return &hclsyntax.LiteralValueExpr{Val: cv, SrcRange: rng}
}

func callToCty(ref map[string]any, node *Tree, funcs map[string]any, u *hclsyntax.FunctionCallExpr) (cty.Value, error) {
	if u.Name == "" {
		return cty.EmptyObjectVal, fmt.Errorf("function call is empty")
	}
	if funcs == nil {
		return cty.EmptyObjectVal, fmt.Errorf("function call is nil for %s", u.Name)
	}
	fn, ok := funcs[u.Name]
	if !ok {
		return cty.EmptyObjectVal, fmt.Errorf("function call is not found for %s", u.Name)
	}
	k := -1
	f := reflect.ValueOf(fn)

	n := 0
	var in []reflect.Value
	for _, arg := range u.Args {
		cv, err := ExpressionToCty(ref, node, arg)
		if err != nil {
			return cty.EmptyObjectVal, err
		}
		v, err := CtyToNative(cv)
		if err != nil {
			return cty.EmptyObjectVal, err
		}
		in = append(in, reflect.ValueOf(v))
		n++
	}
	if f.Type().NumIn() != n {
		return cty.EmptyObjectVal, fmt.Errorf("function %s needs %d args, got %d", u.Name, f.Type().NumIn(), n)
	}

	outputs := f.Call(in)
	m := len(outputs)
	for i, output := range outputs {
		if output.Type().Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			k = i
			break
		}
	}

	if k >= 0 {
		if k >= m {
			return cty.EmptyObjectVal, fmt.Errorf("error output is missing or mismatched")
		}
		if outputs[k].IsValid() && !outputs[k].IsNil() {
			return cty.EmptyObjectVal, outputs[k].Interface().(error)
		}
		if m == 1 {
			return cty.EmptyObjectVal, nil
		}
	}

	for i, output := range outputs {
		if i == k {
			continue
		}
		// in this version, we assume only one item returned
		return NativeToCty(output.Interface())
	}

	return cty.EmptyObjectVal, nil
}

// ExpressionToCty evaluates an HCL expression to a cty.Value.
//
// This function handles:
//   - Tuple/array expressions
//   - Object/map expressions
//   - Variable references (using Tree context)
//   - Function calls (using ref[FUNCTIONS])
//   - Literal values
//
// Parameters:
//   - ref: Context map containing ATTRIBUTES (Tree) and FUNCTIONS
//   - node: Current tree node for variable scope
//   - v: HCL expression to evaluate
//
// Returns the evaluated cty.Value or an error if evaluation fails.
func ExpressionToCty(ref map[string]any, node *Tree, v hclsyntax.Expression) (cty.Value, error) {
	if v == nil {
		return cty.NilVal, nil
	}

	switch t := v.(type) {
	case *hclsyntax.TupleConsExpr:
		var u []cty.Value
		for _, item := range t.Exprs {
			cv, err := ExpressionToCty(ref, node, item)
			if err != nil {
				return cty.EmptyObjectVal, err
			}
			u = append(u, cv)
		}
		return cty.TupleVal(u), nil
	case *hclsyntax.ObjectConsExpr:
		var u = make(map[string]cty.Value)
		for _, item := range t.Items {
			key, err := ExpressionToCty(ref, node, item.KeyExpr)
			if err != nil {
				return cty.EmptyObjectVal, err
			}
			val, err := ExpressionToCty(ref, node, item.ValueExpr)
			if err != nil {
				return cty.EmptyObjectVal, err
			}
			u[key.AsString()] = val
		}
		return cty.ObjectVal(u), nil
	default:
	}

	ctx := new(hcl.EvalContext)
	if ref != nil && ref[ATTRIBUTES] != nil {
		ctx.Variables = CtyVariables(ref[ATTRIBUTES].(*Tree))
	}

	if ref != nil && ref[FUNCTIONS] != nil {
		if u, ok := v.(*hclsyntax.FunctionCallExpr); ok {
			if u.Name == "null" {
				return cty.NilVal, nil
			}
			if ref[FUNCTIONS] == nil {
				return cty.EmptyObjectVal, fmt.Errorf("function call is nil for %s", u.Name)
			}
			switch t := ref[FUNCTIONS].(type) {
			case map[string]function.Function:
				ctx.Functions = t
			case map[string]any:
				return callToCty(ref, node, t, u)
			default:
				return cty.EmptyObjectVal, fmt.Errorf("function call is not a map for %s", u.Name)
			}
		}
	}

	cv, diags := v.Value(ctx)
	if diags.HasErrors() {
		return cty.EmptyObjectVal, (diags.Errs())[0]
	}
	return cv, nil
}

// NativeToCty converts a Go native value to a cty.Value.
//
// Handles conversion of:
//   - map[string]any → cty.Object
//   - []any → cty.Tuple
//   - Primitive types (string, number, bool, etc.)
//
// This is the inverse of CtyToNative.
func NativeToCty(item any) (cty.Value, error) {
	if item == nil {
		return cty.EmptyObjectVal, nil
	}

	switch t := item.(type) {
	case map[string]any:
		hash := make(map[string]cty.Value)
		for k, v := range t {
			ct, err := NativeToCty(v)
			if err != nil {
				return cty.EmptyObjectVal, err
			}
			hash[k] = ct
		}
		return cty.ObjectVal(hash), nil
	case []any:
		var arr []cty.Value
		for _, v := range t {
			ct, err := NativeToCty(v)
			if err != nil {
				return cty.EmptyObjectVal, err
			}
			arr = append(arr, ct)
		}
		return cty.TupleVal(arr), nil
	default:
	}
	typ, err := gocty.ImpliedType(item)
	if err != nil {
		return cty.EmptyObjectVal, err
	}
	return gocty.ToCtyValue(item, typ)
}

func CtyNumberToNative(val cty.Value) (any, error) {
	v := val.AsBigFloat()
	if _, accuracy := v.Int64(); accuracy == big.Exact || accuracy == big.Above {
		var x int64
		err := gocty.FromCtyValue(val, &x)
		if x > 0x7FFFFFFF || x < -0x80000000 {
			return x, err
		}
		return int(x), err
	} else if _, accuracy := v.Int(nil); accuracy == big.Exact || accuracy == big.Above {
		var x int
		err := gocty.FromCtyValue(val, &x)
		return x, err
	} else if _, accuracy := v.Float32(); accuracy == big.Exact || accuracy == big.Above {
		var x float32
		err := gocty.FromCtyValue(val, &x)
		return x, err
	}
	var x float64
	err := gocty.FromCtyValue(val, &x)
	return x, err
}

// CtyToNative converts a cty.Value to a Go native type.
//
// Conversion rules:
//   - cty.String → string
//   - cty.Number → int, int64, float32, or float64 (auto-detected)
//   - cty.Bool → bool
//   - cty.Object/Map → map[string]any
//   - cty.List/Tuple/Set → []any
//   - cty.Null → nil
//
// Numbers are intelligently converted to the smallest type that fits.
// This is the inverse of NativeToCty.
func CtyToNative(val cty.Value) (any, error) {
	if val.IsNull() {
		return nil, nil
	}

	ty := val.Type()
	switch ty {
	case cty.String:
		var v string
		err := gocty.FromCtyValue(val, &v)
		return v, err
	case cty.Number:
		return CtyNumberToNative(val)
	case cty.Bool:
		var v bool
		err := gocty.FromCtyValue(val, &v)
		return v, err
	default:
	}

	switch {
	case ty.IsObjectType(), ty.IsMapType():
		var u map[string]any
		for k, v := range val.AsValueMap() {
			x, err := CtyToNative(v)
			if err != nil {
				return nil, err
			}
			if x == nil {
				continue
			}
			if u == nil {
				u = make(map[string]any)
			}
			u[k] = x
		}
		return u, nil
	case ty.IsListType(), ty.IsTupleType(), ty.IsSetType():
		var u []any
		for _, v := range val.AsValueSlice() {
			x, err := CtyToNative(v)
			if err != nil {
				return nil, err
			}
			if x == nil {
				continue
			}
			u = append(u, x)
		}
		return u, nil
	default:
	}

	return nil, fmt.Errorf("assumed primitive value %#v not implementned", val)
}

// ConvertCtyToFieldType converts a cty.Value to a specific Go type using reflection.
// This function performs type-safe conversion from HCL's cty.Value to any Go type,
// handling all numeric type variants correctly.
//
// Unlike CtyToNative which returns generic types (int, float64), this function
// converts to the exact target type (uint16, int32, float32, etc.).
//
// The function handles type coercion for common mismatches:
//   - cty.Object → map[string]T (e.g., from HCL comprehensions)
//   - cty.Tuple → []T (when tuple and list are semantically equivalent)
//
// Parameters:
//   - ctyVal: the cty.Value to convert
//   - targetType: the reflect.Type to convert to
//
// Returns the converted value as any or an error if conversion fails.
//
// Examples:
//   - ctyVal=5, targetType=uint16 → uint16(5)
//   - ctyVal=3.14, targetType=float32 → float32(3.14)
//   - ctyVal="hello", targetType=string → "hello"
//   - ctyVal=cty.Object(...), targetType=map[string]string → map[string]string{...}
func ConvertCtyToFieldType(ctyVal cty.Value, targetType reflect.Type) (any, error) {
	if ctyVal.IsNull() {
		return reflect.Zero(targetType).Interface(), nil
	}

	// Handle type coercion for common HCL patterns
	// Convert number to string if needed (e.g., from function return values)
	if targetType.Kind() == reflect.String && ctyVal.Type() == cty.Number {
		convertedVal, err := convert.Convert(ctyVal, cty.String)
		if err == nil {
			ctyVal = convertedVal
		}
	}

	// 1. If target is a map and value is an object, convert object to map
	if targetType.Kind() == reflect.Map && ctyVal.Type().IsObjectType() {
		// Get the element type of the map
		elemType := targetType.Elem()

		// Determine the target cty type for map elements
		var targetCtyElemType cty.Type
		if elemType.Kind() == reflect.String {
			targetCtyElemType = cty.String
		} else if elemType.Kind() == reflect.Slice && elemType.Elem().Kind() == reflect.String {
			// Handle map[string][]string case
			targetCtyElemType = cty.List(cty.String)
		} else {
			targetCtyElemType = cty.DynamicPseudoType
		}

		// Convert object to map
		convertedVal, err := convert.Convert(ctyVal, cty.Map(targetCtyElemType))
		if err == nil {
			ctyVal = convertedVal
		}
	}

	// 2. If target is a slice and value is a tuple, convert tuple to list
	if targetType.Kind() == reflect.Slice && ctyVal.Type().IsTupleType() {
		// Get the element type of the slice
		elemType := targetType.Elem()

		// Build a list type
		var listType cty.Type
		if elemType.Kind() == reflect.String {
			listType = cty.List(cty.String)
		} else {
			listType = cty.List(cty.DynamicPseudoType)
		}

		// Try to convert the tuple to a list
		convertedVal, err := convert.Convert(ctyVal, listType)
		if err == nil {
			ctyVal = convertedVal
		}
	}

	// Create a pointer to a new zero value of the target type
	targetPtr := reflect.New(targetType).Interface()

	// Use gocty to convert with exact type matching
	err := gocty.FromCtyValue(ctyVal, targetPtr)
	if err != nil {
		return nil, fmt.Errorf("failed to convert cty.Value to %v: %w", targetType, err)
	}

	// Dereference the pointer to get the actual value
	return reflect.ValueOf(targetPtr).Elem().Interface(), nil
}

// CtyVariables converts a Tree's variables to cty.Value format for HCL expression evaluation.
//
// This function recursively traverses the tree and converts all stored values to cty.Value.
// Values stored in the tree Data map are expected to already be cty.Value.
//
// Parameters:
//   - tree: The tree node to extract variables from
//
// Returns a map suitable for use as HCL EvalContext.Variables.
func CtyVariables(tree *Tree) map[string]cty.Value {
	hash := make(map[string]cty.Value)

	// Lock and copy children to avoid holding lock during recursion
	tree.mu.RLock()
	downs := make([]*Tree, len(tree.Downs))
	copy(downs, tree.Downs)
	name := tree.Name
	tree.mu.RUnlock()

	for _, down := range downs {
		if variables := CtyVariables(down); variables != nil {
			hash[down.Name] = cty.ObjectVal(variables)
		}
	}

	// Data.Range is already thread-safe (sync.Map)
	tree.Data.Range(func(k, v any) bool {
		if k != VAR {
			// Values should already be cty.Value
			if cv, ok := v.(cty.Value); ok {
				hash[k.(string)] = cv
			}
		}
		return true
	})

	if name == VAR {
		hash[VAR] = cty.ObjectVal(hash)
	}

	return hash
}

// NewTreeCtyFunction returns the default tree and a map containing default cty functions.
//
// This function initializes or reuses a Tree node for HCL variable context and sets up
// the default cty function library for HCL expression evaluation.
//
// Parameters:
//   - ref: Context map. If nil, a new map will be created.
//
// The ref map is modified to include:
//   - ref[ATTRIBUTES]: A Tree node for variable storage (created if not present)
//   - ref[FUNCTIONS]: Default cty functions from ilang.CoreFunctions (merged if already present)
//
// Returns:
//   - *Tree: The Tree node for storing variables (from ref[ATTRIBUTES])
//   - map[string]any: The ref map with both ATTRIBUTES and FUNCTIONS set
//
// Usage:
//
//	node, ref := NewTreeCtyFunction(nil)
//	// Now ref[ATTRIBUTES] contains the tree and ref[FUNCTIONS] contains cty functions
func NewTreeCtyFunction(ref map[string]any) (*Tree, map[string]any) {
	if ref == nil {
		ref = make(map[string]any)
	}
	var node *Tree
	if inode, ok := ref[ATTRIBUTES]; ok {
		node = inode.(*Tree)
	} else {
		node = NewTree(VAR)
		ref[ATTRIBUTES] = node
	}
	defaultFuncs := ilang.CoreFunctions(".")
	if ref[FUNCTIONS] == nil {
		ref[FUNCTIONS] = defaultFuncs
	} else if t, ok := ref[FUNCTIONS].(map[string]function.Function); ok {
		for k, v := range defaultFuncs {
			t[k] = v
		}
	}

	return node, ref
}

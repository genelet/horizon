package dethcl

import (
	"fmt"
	"reflect"

	"github.com/genelet/horizon/utils"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// processBlockFields handles complex block fields based on their spec type.
// This includes Map2Struct, MapStruct, ListStruct, and SingleStruct.
func processBlockFields(node *utils.Tree, file *hcl.File, ref map[string]any, oriFields []reflect.StructField, oriblock map[string][]*hclsyntax.Block, objectMap map[string]*utils.Value, oriTobe reflect.Value) error {
	for _, field := range oriFields {
		tag := (parseHCLTag(field.Tag))[0]
		blocks := oriblock[tag]
		if len(blocks) == 0 {
			continue
		}

		name := field.Name
		result := objectMap[name]

		if result == nil {
			continue
		}

		if x := result.GetMap2Struct(); x != nil {
			if err := processMap2StructField(node, file, ref, field, blocks, x, oriTobe); err != nil {
				return err
			}
		} else if x := result.GetMapStruct(); x != nil {
			if err := processMapStructField(node, file, ref, field, blocks, x, oriTobe); err != nil {
				return err
			}
		} else if x := result.GetListStruct(); x != nil {
			if err := processListStructField(node, file, ref, field, blocks, x, oriTobe); err != nil {
				return err
			}
		} else if x := result.GetSingleStruct(); x != nil {
			if err := processSingleStructField(node, file, ref, field, blocks[0], x, oriTobe); err != nil {
				return err
			}
		}
	}
	return nil
}

// processMap2StructField handles fields with Map2Struct spec (map with 2 labels).
func processMap2StructField(node *utils.Tree, file *hcl.File, ref map[string]any, field reflect.StructField, blocks []*hclsyntax.Block, mapSpec *utils.Map2Struct, oriTobe reflect.Value) error {
	name := field.Name
	tag := (parseHCLTag(field.Tag))[0]
	typ := field.Type

	if typ.Kind() != reflect.Map {
		return fmt.Errorf("field %s: expected map type for Map2Struct, got %v", name, typ.Kind())
	}

	nextMap2Structs := mapSpec.GetMap2Fields()
	var first *utils.MapStruct
	var firstFirst *utils.Struct
	for _, first = range nextMap2Structs {
		for _, firstFirst = range first.GetMapFields() {
			break
		}
		break
	}

	n := len(blocks)
	fMap := reflect.MakeMapWithSize(typ, n)
	f := oriTobe.Elem().FieldByName(name)

	for k := 0; k < n; k++ {
		block := blocks[k]
		subnode := node.GetNode(tag, block.Labels...)

		var keystring0, keystring1 string
		if len(block.Labels) > 0 {
			keystring0 = block.Labels[0]
			if len(block.Labels) > 1 {
				keystring1 = block.Labels[1]
			}
		}

		nextMapStruct, ok := nextMap2Structs[keystring0]
		if !ok {
			nextMapStruct = first
		}
		nextStruct, ok := nextMapStruct.GetMapFields()[keystring1]
		if !ok {
			nextStruct = firstFirst
		}

		trial := ref[nextStruct.ClassName]
		if trial == nil {
			return fmt.Errorf("field %s: struct type %q not found in ref map", name, nextStruct.ClassName)
		}
		trial = clone(trial)

		s, lbls, err := getBlockBytes(block, file)
		if err != nil {
			return err
		}
		if len(lbls) > 2 {
			return fmt.Errorf("field %s: Map2Struct supports maximum 2 labels, got %d", name, len(lbls))
		}

		err = tryUnmarshalWithCustom(subnode, s, trial, nextStruct, ref, lbls...)
		if err != nil {
			return fmt.Errorf("field %s[%s][%s]: unmarshal failed: %w", name, keystring0, keystring1, err)
		}

		// Get labels from struct if not in HCL
		key0, key1 := getLabels(trial)
		if keystring0 == "" {
			keystring0 = key0
			keystring1 = key1
		} else if keystring1 == "" {
			if key1 != "" {
				keystring1 = key1
			} else if key0 != "" {
				keystring1 = key0
			}
		}
		strKey := reflect.ValueOf([2]string{keystring0, keystring1})

		knd := typ.Elem().Kind()
		if knd == reflect.Interface || knd == reflect.Ptr {
			fMap.SetMapIndex(strKey, reflect.ValueOf(trial))
		} else {
			fMap.SetMapIndex(strKey, reflect.ValueOf(trial).Elem())
		}
	}
	f.Set(fMap)
	return nil
}

// processMapStructField handles fields with MapStruct spec (map with 1 label).
func processMapStructField(node *utils.Tree, file *hcl.File, ref map[string]any, field reflect.StructField, blocks []*hclsyntax.Block, mapSpec *utils.MapStruct, oriTobe reflect.Value) error {
	name := field.Name
	tag := (parseHCLTag(field.Tag))[0]
	typ := field.Type

	if typ.Kind() != reflect.Map {
		return fmt.Errorf("field %s: expected map type for MapStruct, got %v", name, typ.Kind())
	}

	nextMapStructs := mapSpec.GetMapFields()
	var first *utils.Struct
	for _, first = range nextMapStructs {
		break
	}

	n := len(blocks)
	fMap := reflect.MakeMapWithSize(typ, n)
	f := oriTobe.Elem().FieldByName(name)

	for k := 0; k < n; k++ {
		block := blocks[k]
		subnode := node.GetNode(tag, block.Labels...)
		keystring := block.Labels[0]

		nextStruct, ok := nextMapStructs[keystring]
		if !ok {
			nextStruct = first
		}

		trial := ref[nextStruct.ClassName]
		if trial == nil {
			return fmt.Errorf("field %s: struct type %q not found in ref map", name, nextStruct.ClassName)
		}
		trial = clone(trial)

		s, lbls, err := getBlockBytes(block, file)
		if err != nil {
			return err
		}
		if len(lbls) > 1 {
			return fmt.Errorf("field %s: MapStruct supports maximum 1 label, got %d", name, len(lbls))
		}

		err = tryUnmarshalWithCustom(subnode, s, trial, nextStruct, ref, lbls...)
		if err != nil {
			return fmt.Errorf("field %s[%s]: unmarshal failed: %w", name, keystring, err)
		}

		knd := typ.Elem().Kind()
		strKey := reflect.ValueOf(keystring)

		if knd == reflect.Interface || knd == reflect.Ptr {
			fMap.SetMapIndex(strKey, reflect.ValueOf(trial))
		} else {
			fMap.SetMapIndex(strKey, reflect.ValueOf(trial).Elem())
		}
	}
	f.Set(fMap)
	return nil
}

// processListStructField handles fields with ListStruct spec (slice or map without labels).
func processListStructField(node *utils.Tree, file *hcl.File, ref map[string]any, field reflect.StructField, blocks []*hclsyntax.Block, listSpec *utils.ListStruct, oriTobe reflect.Value) error {
	name := field.Name
	tag := (parseHCLTag(field.Tag))[0]
	typ := field.Type
	f := oriTobe.Elem().FieldByName(name)

	nextListStructs := listSpec.GetListFields()
	nSmaller := len(nextListStructs)
	first := nextListStructs[0]

	n := len(blocks)

	var fSlice, fMap reflect.Value
	if typ.Kind() == reflect.Map {
		fMap = reflect.MakeMapWithSize(typ, n)
	} else {
		fSlice = reflect.MakeSlice(typ, n, n)
	}

	for k := 0; k < n; k++ {
		nextStruct := first
		if k < nSmaller && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) {
			nextStruct = nextListStructs[k]
		}

		block := blocks[k]
		subnode := node.GetNode(tag, block.Labels...)

		trial := ref[nextStruct.ClassName]
		if trial == nil {
			return fmt.Errorf("field %s: struct type %q not found in ref map (list index %d)", name, nextStruct.ClassName, k)
		}
		trial = clone(trial)

		s, lbls, err := getBlockBytes(block, file)
		if err != nil {
			return err
		}

		err = tryUnmarshalWithCustom(subnode, s, trial, nextStruct, ref, lbls...)
		if err != nil {
			return fmt.Errorf("field %s[%d]: unmarshal failed: %w", name, k, err)
		}

		knd := typ.Elem().Kind()
		if typ.Kind() == reflect.Map {
			strKey := reflect.ValueOf(lbls[0])
			if knd == reflect.Interface || knd == reflect.Ptr {
				fMap.SetMapIndex(strKey, reflect.ValueOf(trial))
			} else {
				fMap.SetMapIndex(strKey, reflect.ValueOf(trial).Elem())
			}
		} else {
			if knd == reflect.Interface || knd == reflect.Ptr {
				fSlice.Index(k).Set(reflect.ValueOf(trial))
			} else {
				fSlice.Index(k).Set(reflect.ValueOf(trial).Elem())
			}
		}
	}

	if typ.Kind() == reflect.Map {
		f.Set(fMap)
	} else {
		f.Set(fSlice)
	}
	return nil
}

// processSingleStructField handles fields with SingleStruct spec (single nested struct).
func processSingleStructField(node *utils.Tree, file *hcl.File, ref map[string]any, field reflect.StructField, block *hclsyntax.Block, singleSpec *utils.Struct, oriTobe reflect.Value) error {
	name := field.Name
	tag := (parseHCLTag(field.Tag))[0]
	f := oriTobe.Elem().FieldByName(name)

	subnode := node.GetNode(tag, block.Labels...)
	trial := ref[singleSpec.ClassName]
	if trial == nil {
		return fmt.Errorf("field %s: struct type %q not found in ref map", name, singleSpec.ClassName)
	}
	trial = clone(trial)

	s, lbls, err := getBlockBytes(block, file)
	if err != nil {
		return err
	}

	err = tryUnmarshalWithCustom(subnode, s, trial, singleSpec, ref, lbls...)
	if err != nil {
		return fmt.Errorf("field %s: unmarshal failed: %w", name, err)
	}

	if f.Kind() == reflect.Interface || f.Kind() == reflect.Ptr {
		f.Set(reflect.ValueOf(trial))
	} else {
		f.Set(reflect.ValueOf(trial).Elem())
	}
	return nil
}

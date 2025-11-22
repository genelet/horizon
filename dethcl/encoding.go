package dethcl

import (
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/genelet/horizon/utils"
)

// mapStructureType represents different types of map structures.
type mapStructureType int

const (
	notAMap    mapStructureType = 0 // Not a map
	shallowMap mapStructureType = 1 // Map with non-map values
	nestedMap  mapStructureType = 2 // Map with all values being maps
)

// classifyMapStructure determines if an item is a map and what type of map it is.
// Returns the map type and the map itself (nil if not a map).
func classifyMapStructure(item any) (mapStructureType, map[string]any) {
	m, ok := item.(map[string]any)
	if !ok {
		return notAMap, nil
	}

	if len(m) == 0 {
		return shallowMap, m
	}

	// Check if all values are maps
	allMaps := true
	for _, v := range m {
		if _, ok := v.(map[string]any); !ok {
			allMaps = false
			break
		}
	}

	if allMaps {
		return nestedMap, m
	}
	return shallowMap, m
}

// isHashAll is deprecated. Use classifyMapStructure instead.
// Maintained for backward compatibility.
func isHashAll(item any) (int, map[string]any) {
	typ, m := classifyMapStructure(item)
	return int(typ), m
}

// encodePrimitiveOrRecurse attempts to encode a value as a primitive (string, bool, number).
// If the value is complex, it returns the bytes from recursive marshaling.
// Returns: (primitiveString, recursiveBytes, error)
// - If primitiveString != "", use that (it's a simple value)
// - If recursiveBytes != nil, use that (it's a complex value)
func encodePrimitiveOrRecurse(item any, equal bool, level int) (string, []byte, error) {
	switch item.(type) {
	case string:
		return fmt.Sprintf("\"%s\"", item), nil, nil
	case bool:
		return fmt.Sprintf("%t", item), nil, nil
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", item), nil, nil
	case float32, float64:
		c, err := utils.NativeToCty(item)
		if err != nil {
			return "", nil, err
		}
		n, err := utils.CtyNumberToNative(c)
		if err != nil {
			return "", nil, err
		}
		switch n.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return fmt.Sprintf("%d", n), nil, nil
		default:
		}
		return fmt.Sprintf("%f", n), nil, nil
	default:
	}

	bs, err := marshalLevel(item, equal, level+1)
	return "", bs, err
}

func loopHash(lines *[]string, header string, item any, equal bool, depth, level int, keyname ...string) error {
	mapType, nextMap := classifyMapStructure(item)

	// Limit HCL labels to 2. If deeper, treat as block body.
	if depth >= 2 && mapType == nestedMap {
		mapType = shallowMap
	}

	switch mapType {
	case nestedMap:
		// Sort keys for deterministic output
		keys := make([]string, 0, len(nextMap))
		for k := range nextMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			value := nextMap[key]
			nextHeader := header + ` "` + key + `"`
			err := loopHash(lines, nextHeader, value, false, depth+1, level)
			if err != nil {
				return err
			}
		}
	case shallowMap:
		// pass 'header' as the keyname to the next 'default' below
		bs, err := marshalLevel(item, equal, level+1, header)
		if err != nil {
			return err
		}
		*lines = append(*lines, fmt.Sprintf("%s %s", header, bs))
	default:
		str, bs, err := encodePrimitiveOrRecurse(item, equal, level)
		if err != nil {
			return err
		}
		if bs != nil && len(keyname) > 0 && matchlast(keyname[0], string(bs)) {
			return nil
		}
		if str != "" {
			*lines = append(*lines, fmt.Sprintf("%s = %s", header, str))
		} else {
			*lines = append(*lines, fmt.Sprintf("%s = %s", header, bs))
		}
	}
	return nil
}

func matchlast(keyname string, name string) bool {
	names := strings.Split(keyname, " ")
	keyname = names[len(names)-1]
	return keyname == name
}

func encoding(current any, equal bool, level int, keyname ...string) ([]byte, error) {
	var str string
	if current == nil {
		return nil, nil
	}

	rv := reflect.ValueOf(current)
	switch rv.Kind() {
	case reflect.Struct:
		return marshalLevel(current, false, level, keyname...)
	case reflect.Pointer:
		return marshalLevel(rv.Elem().Interface(), equal, level, keyname...)
	case reflect.Map:
		return encodeMap(rv, equal, level, keyname...)
	case reflect.Slice, reflect.Array:
		return encodeSlice(rv, level)
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr, reflect.Float32, reflect.Float64:
		var err error
		str, _, err = encodePrimitiveOrRecurse(current, equal, level)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("data type %v not supported", rv.Kind())
	}

	return []byte(str), nil
}

func encodeMap(rv reflect.Value, equal bool, level int, keyname ...string) ([]byte, error) {
	var arr []string
	iter := rv.MapRange()
	for iter.Next() {
		key := iter.Key()
		if key.Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string, got %v", key.Kind())
		}
		switch iter.Value().Kind() {
		case reflect.Interface, reflect.Ptr, reflect.Map, reflect.Slice, reflect.Func:
			if iter.Value().IsNil() {
				arr = append(arr, fmt.Sprintf("%s = null()", key.String()))
				continue
			}
		default:
		}
		if len(keyname) > 0 && keyname[0] == markerNoBrackets {
			str, bs, err := encodePrimitiveOrRecurse(iter.Value().Interface(), equal, level)
			if err != nil {
				return nil, err
			}
			if str != "" {
				arr = append(arr, fmt.Sprintf("%s = %s", key.String(), str))
			} else {
				arr = append(arr, fmt.Sprintf("%s = %s", key.String(), bs))
			}
		} else {
			err := loopHash(&arr, key.String(), iter.Value().Interface(), equal, 0, level, keyname...)
			if err != nil {
				return nil, err
			}
		}
	}

	leading := strings.Repeat("  ", level+1)
	lessLeading := strings.Repeat("  ", level)
	var str string
	if level == 0 {
		str = fmt.Sprintf("\n%s\n%s", leading+strings.Join(arr, "\n"+leading), lessLeading)
	} else {
		str = fmt.Sprintf("{\n%s\n%s}", leading+strings.Join(arr, "\n"+leading), lessLeading)
	}
	return []byte(str), nil
}

func encodeSlice(rv reflect.Value, level int) ([]byte, error) {
	var arr []string
	for i := 0; i < rv.Len(); i++ {
		bs, err := marshalLevel(rv.Index(i).Interface(), true, level+1, markerNoBrackets)
		if err != nil {
			return nil, err
		}
		item := `[]`
		if bs != nil {
			item = string(bs)
		}
		arr = append(arr, item)
	}

	leading := strings.Repeat("  ", level+1)
	lessLeading := strings.Repeat("  ", level)
	str := fmt.Sprintf("[\n%s\n%s]", leading+strings.Join(arr, ",\n"+leading), lessLeading)
	return []byte(str), nil
}

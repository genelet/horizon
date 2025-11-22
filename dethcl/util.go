package dethcl

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
)

// clone creates a shallow copy of a struct value by copying all settable fields.
// The input must be a pointer to a struct.
// Returns a new pointer to a struct of the same type with copied field values.
func clone(old interface{}) interface{} {
	obj := reflect.New(reflect.TypeOf(old).Elem())
	oldVal := reflect.ValueOf(old).Elem()
	newVal := obj.Elem()
	for i := 0; i < oldVal.NumField(); i++ {
		newValField := newVal.Field(i)
		if newValField.CanSet() {
			newValField.Set(oldVal.Field(i))
		}
	}

	return obj.Interface()
}

// parseHCLTag extracts the HCL tag name and modifier from a struct field tag.
// Returns [0] = tag name, [1] = modifier (e.g., "label", "block", "optional")
// Example: `hcl:"name,label"` returns ["name", "label"]
func parseHCLTag(tag reflect.StructTag) [2]string {
	for _, tagStr := range strings.Fields(string(tag)) {
		if len(tagStr) >= TagPrefixHCLLength && strings.ToLower(tagStr[:TagPrefixHCLLength]) == TagPrefixHCL {
			tagStr = tagStr[TagPrefixHCLLength : len(tagStr)-1]
			parts := strings.SplitN(tagStr, ",", 2)
			if len(parts) == 2 {
				return [2]string{parts[0], parts[1]}
			}
			return [2]string{parts[0], ""}
		}
	}
	return [2]string{}
}

// extractHCLTagName returns just the HCL tag name (without modifier) as bytes.
func extractHCLTagName(tag reflect.StructTag) []byte {
	parsed := parseHCLTag(tag)
	return []byte(parsed[0])
}

// generateTempHCLFileName creates a random temporary HCL filename.
// Used internally for parsing HCL fragments that don't have a source file.
// Format: <random-number>.hcl
func generateTempHCLFileName() string {
	return fmt.Sprintf("%d%s", rand.Int(), HCLFileExtension)
}

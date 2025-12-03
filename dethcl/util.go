package dethcl

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"strings"

	"github.com/genelet/horizon/utils"
)

// clone creates a new zero-value instance of the same type as old.
// The old parameter must be a pointer to a struct.
// This is used to create fresh instances from type registry templates
// before unmarshaling HCL data into them.
//
// Note: This creates a zero-value instance rather than copying fields.
// This is intentional because:
// 1. Templates in the type registry should be zero-value prototypes
// 2. The instance will be immediately populated by HCL unmarshaling
// 3. This avoids shallow copy issues with pointer fields
func clone(old any) any {
	return reflect.New(reflect.TypeOf(old).Elem()).Interface()
}

// parseHCLTag extracts the HCL tag name and modifier from a struct field tag.
// Returns [0] = tag name, [1] = modifier (e.g., "label", "block", "optional")
// Example: `hcl:"name,label"` returns ["name", "label"]
func parseHCLTag(tag reflect.StructTag) [2]string {
	for _, tagStr := range strings.Fields(string(tag)) {
		if len(tagStr) >= tagPrefixHCLLength && strings.ToLower(tagStr[:tagPrefixHCLLength]) == tagPrefixHCL {
			tagStr = tagStr[tagPrefixHCLLength : len(tagStr)-1]
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
	return fmt.Sprintf("%d%s", rand.Int(), hclFileExtension)
}

// getFirstStructFromMap returns the first struct from a map using deterministic key order.
// This ensures consistent behavior across runs since Go map iteration order is random.
func getFirstStructFromMap(m map[string]*utils.Struct) *utils.Struct {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return m[keys[0]]
}

// getFirstMapStructFromMap returns the first MapStruct from a map using deterministic key order.
func getFirstMapStructFromMap(m map[string]*utils.MapStruct) *utils.MapStruct {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return m[keys[0]]
}

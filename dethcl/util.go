package dethcl

import (
	"cmp"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/OpenUdon/schema"
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

// sortedStringKeys returns the keys of a string-keyed map in ascending
// sorted order. Used wherever we need deterministic iteration order over a
// string-keyed map, since Go map iteration order is randomized.
func sortedStringKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// getFirstStructFromMap returns the first struct from a map using deterministic key order.
// This ensures consistent behavior across runs since Go map iteration order is random.
func getFirstStructFromMap(m map[string]*schema.Struct) *schema.Struct {
	if len(m) == 0 {
		return nil
	}
	return m[sortedStringKeys(m)[0]]
}

// getFirstMapStructFromMap returns the first MapStruct from a map using deterministic key order.
func getFirstMapStructFromMap(m map[string]*schema.MapStruct) *schema.MapStruct {
	if len(m) == 0 {
		return nil
	}
	return m[sortedStringKeys(m)[0]]
}

// mapKeyLabels converts a reflect.Value map key into the HCL block labels
// (or sort key) it represents. Interface keys are unwrapped to their
// concrete value first. Array/slice keys become one label per element
// (trailing zero-value elements dropped); every other kind becomes a
// single label from key.String() — meaningful only for string-kind keys,
// since reflect.Value.String() returns a fixed placeholder for other kinds
// (e.g. int). Non-string, non-array key kinds are a pre-existing, narrower
// limitation (placeholder label text); sortedMapKeys below only guarantees
// their relative order is stable, not that the label text is meaningful.
func mapKeyLabels(key reflect.Value) []string {
	if key.Kind() == reflect.Interface {
		key = key.Elem()
	}
	if key.Kind() != reflect.Array && key.Kind() != reflect.Slice {
		return []string{key.String()}
	}

	labels := make([]string, 0, key.Len())
	for i := 0; i < key.Len(); i++ {
		item := key.Index(i)
		if !item.IsZero() {
			labels = append(labels, item.String())
		}
	}
	return labels
}

// mapKeyEntry pairs a map key with its precomputed labels, so sorting and
// the caller's iteration don't each recompute mapKeyLabels for every key.
type mapKeyEntry struct {
	key    reflect.Value
	labels []string
}

// sortedMapKeys returns a map's keys in deterministic order: primarily by
// mapKeyLabels, and secondarily by the key's own %v formatting. A pure
// label comparison isn't a total order — distinct keys can produce
// identical or empty labels (e.g. [2]string{"a",""} vs {"","a"}, or any
// non-string/array key kind) — so the %v tie-breaker ensures the order
// depends only on the key's value, never on Go's randomized map iteration.
func sortedMapKeys(value reflect.Value) []mapKeyEntry {
	rawKeys := value.MapKeys()
	entries := make([]mapKeyEntry, len(rawKeys))
	for i, k := range rawKeys {
		entries[i] = mapKeyEntry{key: k, labels: mapKeyLabels(k)}
	}
	slices.SortFunc(entries, func(a, b mapKeyEntry) int {
		if c := slices.Compare(a.labels, b.labels); c != 0 {
			return c
		}
		return cmp.Compare(fmt.Sprintf("%v", a.key.Interface()), fmt.Sprintf("%v", b.key.Interface()))
	})
	return entries
}

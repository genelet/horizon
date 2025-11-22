package convert

import (
	"encoding/json"
	"fmt"

	"github.com/genelet/horizon/dethcl"
	"gopkg.in/yaml.v3"
)

// UnmarshalFunc is a function that unmarshals data into a target object.
type UnmarshalFunc func([]byte, any) error

// MarshalFunc is a function that marshals an object into bytes.
type MarshalFunc func(any) ([]byte, error)

// convertFormat is a generic converter that unmarshals from one format and marshals to another.
// This eliminates code duplication across all conversion functions.
func convertFormat(raw []byte, unmarshal UnmarshalFunc, marshal MarshalFunc) ([]byte, error) {
	if len(raw) == 0 {
		return nil, fmt.Errorf("input is empty")
	}

	obj := map[string]any{}
	if err := unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}
	result, err := marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal output: %w", err)
	}
	return result, nil
}

// hclUnmarshal wraps dethcl.Unmarshal to match the UnmarshalFunc signature.
// dethcl.Unmarshal has variadic labels parameter which we don't need for format conversion.
func hclUnmarshal(data []byte, v any) error {
	return dethcl.Unmarshal(data, v)
}

// JSONToYAML converts JSON data to YAML format.
//
// The conversion is lossless for basic data types (strings, numbers, bools,
// arrays, and objects). The JSON is first unmarshaled to a generic map
// structure, then marshaled to YAML.
//
// Returns the YAML-formatted data or an error if parsing or conversion fails.
func JSONToYAML(raw []byte) ([]byte, error) {
	return convertFormat(raw, json.Unmarshal, yaml.Marshal)
}

// YAMLToJSON converts YAML data to JSON format.
//
// Returns the JSON-formatted data or an error if parsing or conversion fails.
func YAMLToJSON(raw []byte) ([]byte, error) {
	return convertFormat(raw, yaml.Unmarshal, json.Marshal)
}

// JSONToHCL converts JSON data to HCL format.
//
// Note: The HCL output will not contain variables or expressions, only
// declarative data. Maps are represented as HCL blocks with labels.
//
// Returns the HCL-formatted data or an error if parsing or conversion fails.
func JSONToHCL(raw []byte) ([]byte, error) {
	return convertFormat(raw, json.Unmarshal, dethcl.Marshal)
}

// HCLToJSON converts HCL data to JSON format.
//
// Important: The HCL input should not contain variables or complex expressions,
// only declarative data structures. Such features will cause errors.
//
// Returns the JSON-formatted data or an error if parsing or conversion fails.
func HCLToJSON(raw []byte) ([]byte, error) {
	return convertFormat(raw, hclUnmarshal, json.Marshal)
}

// YAMLToHCL converts YAML data to HCL format.
//
// Note: The HCL output will not contain variables or expressions, only
// declarative data. Maps are represented as HCL blocks with labels.
//
// Returns the HCL-formatted data or an error if parsing or conversion fails.
func YAMLToHCL(raw []byte) ([]byte, error) {
	return convertFormat(raw, yaml.Unmarshal, dethcl.Marshal)
}

// HCLToYAML converts HCL data to YAML format.
//
// Important: The HCL input should not contain variables or complex expressions,
// only declarative data structures. Such features will cause errors.
//
// Returns the YAML-formatted data or an error if parsing or conversion fails.
func HCLToYAML(raw []byte) ([]byte, error) {
	return convertFormat(raw, hclUnmarshal, yaml.Marshal)
}

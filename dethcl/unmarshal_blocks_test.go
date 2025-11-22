package dethcl

import (
	"reflect"
	"testing"

	"github.com/genelet/horizon/utils"
)

// These are integration-style tests that verify block processing works correctly
// The actual block processing functions (processMap2StructField, processMapStructField, etc.)
// are complex and depend heavily on HCL parsing, Tree structures, and spec definitions.
// They are already extensively tested through the existing integration tests
// (TestMHclShape, TestMHclFrame, TestMHclChild, etc.)

// Test Map2Struct - map with 2 labels
func TestUnmarshalMap2Struct(t *testing.T) {
	type Inner struct {
		Value string `hcl:"value"`
	}

	type Outer struct {
		Type   string               `hcl:"type,label"`
		Name   string               `hcl:"name,label"`
		Nested map[[2]string]*Inner `hcl:"nested"`
	}

	hclData := []byte(`
		type = "test"
		name = "example"

		nested "key1" "key2" {
			value = "nested_value"
		}
	`)

	// Create spec for nested field
	spec, err := utils.NewStruct("Outer",
		map[string]interface{}{
			"Nested": map[[2]string]string{{"", ""}: "Inner"},
		},
	)
	if err != nil {
		t.Fatalf("failed to create spec: %v", err)
	}

	ref := map[string]interface{}{
		"Inner": new(Inner),
	}

	result := &Outer{}
	err = UnmarshalSpec(hclData, result, spec, ref)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.Type != "test" {
		t.Errorf("Type = %q, want %q", result.Type, "test")
	}
	if result.Name != "example" {
		t.Errorf("Name = %q, want %q", result.Name, "example")
	}

	if result.Nested == nil {
		t.Fatal("Nested should not be nil")
	}

	key := [2]string{"key1", "key2"}
	nested, ok := result.Nested[key]
	if !ok {
		t.Fatalf("Nested[%v] not found", key)
	}

	if nested.Value != "nested_value" {
		t.Errorf("Nested[%v].Value = %q, want %q", key, nested.Value, "nested_value")
	}
}

// Test MapStruct - map with 1 label
func TestUnmarshalMapStruct(t *testing.T) {
	type Config struct {
		Value string `hcl:"value"`
	}

	type Root struct {
		Name    string            `hcl:"name"`
		Configs map[string]Config `hcl:"config"`
	}

	hclData := []byte(`
		name = "root"

		config "prod" {
			value = "production"
		}

		config "dev" {
			value = "development"
		}
	`)

	spec, err := utils.NewStruct("Root",
		map[string]interface{}{
			"Configs": map[string]string{"": "Config"},
		},
	)
	if err != nil {
		t.Fatalf("failed to create spec: %v", err)
	}

	ref := map[string]interface{}{
		"Config": new(Config),
	}

	result := &Root{}
	err = UnmarshalSpec(hclData, result, spec, ref)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.Name != "root" {
		t.Errorf("Name = %q, want %q", result.Name, "root")
	}

	if len(result.Configs) != 2 {
		t.Fatalf("len(Configs) = %d, want 2", len(result.Configs))
	}

	if result.Configs["prod"].Value != "production" {
		t.Errorf("Configs[prod].Value = %q, want %q", result.Configs["prod"].Value, "production")
	}
	if result.Configs["dev"].Value != "development" {
		t.Errorf("Configs[dev].Value = %q, want %q", result.Configs["dev"].Value, "development")
	}
}

// Test ListStruct - slice/array
func TestUnmarshalListStruct(t *testing.T) {
	type Item struct {
		Name  string `hcl:"name"`
		Value int    `hcl:"value"`
	}

	type Root struct {
		Title string `hcl:"title"`
		Items []Item `hcl:"item"`
	}

	hclData := []byte(`
		title = "test"

		item {
			name = "first"
			value = 1
		}

		item {
			name = "second"
			value = 2
		}
	`)

	spec, err := utils.NewStruct("Root",
		map[string]interface{}{
			"Items": []string{"Item"},
		},
	)
	if err != nil {
		t.Fatalf("failed to create spec: %v", err)
	}

	ref := map[string]interface{}{
		"Item": new(Item),
	}

	result := &Root{}
	err = UnmarshalSpec(hclData, result, spec, ref)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.Title != "test" {
		t.Errorf("Title = %q, want %q", result.Title, "test")
	}

	if len(result.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(result.Items))
	}

	if result.Items[0].Name != "first" || result.Items[0].Value != 1 {
		t.Errorf("Items[0] = {%q, %d}, want {%q, %d}", result.Items[0].Name, result.Items[0].Value, "first", 1)
	}
	if result.Items[1].Name != "second" || result.Items[1].Value != 2 {
		t.Errorf("Items[1] = {%q, %d}, want {%q, %d}", result.Items[1].Name, result.Items[1].Value, "second", 2)
	}
}

// Test SingleStruct - single nested block
func TestUnmarshalSingleStruct(t *testing.T) {
	type Metadata struct {
		Author  string `hcl:"author"`
		Version string `hcl:"version"`
	}

	type Document struct {
		Title    string   `hcl:"title"`
		Metadata Metadata `hcl:"metadata"`
	}

	hclData := []byte(`
		title = "My Document"

		metadata {
			author = "John Doe"
			version = "1.0"
		}
	`)

	spec, err := utils.NewStruct("Document",
		map[string]interface{}{
			"Metadata": "Metadata",
		},
	)
	if err != nil {
		t.Fatalf("failed to create spec: %v", err)
	}

	ref := map[string]interface{}{
		"Metadata": new(Metadata),
	}

	result := &Document{}
	err = UnmarshalSpec(hclData, result, spec, ref)
	if err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if result.Title != "My Document" {
		t.Errorf("Title = %q, want %q", result.Title, "My Document")
	}

	if result.Metadata.Author != "John Doe" {
		t.Errorf("Metadata.Author = %q, want %q", result.Metadata.Author, "John Doe")
	}
	if result.Metadata.Version != "1.0" {
		t.Errorf("Metadata.Version = %q, want %q", result.Metadata.Version, "1.0")
	}
}

// Test error case - missing struct type in ref map
func TestProcessBlockFieldsErrorMissingRef(t *testing.T) {
	type Inner struct {
		Value string `hcl:"value"`
	}

	type Outer struct {
		Nested Inner `hcl:"nested"`
	}

	hclData := []byte(`
		nested {
			value = "test"
		}
	`)

	spec, err := utils.NewStruct("Outer",
		map[string]interface{}{
			"Nested": "Inner",
		},
	)
	if err != nil {
		t.Fatalf("failed to create spec: %v", err)
	}

	// Deliberately omit "Inner" from ref map
	ref := map[string]interface{}{}

	result := &Outer{}
	err = UnmarshalSpec(hclData, result, spec, ref)

	// Should get an error about missing struct type
	if err == nil {
		t.Error("expected error for missing struct type in ref map, got nil")
	}
}

// Test parseHCLTag helper (used by block processing)
func TestParseHCLTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		expected [2]string
	}{
		{
			name:     "simple tag",
			tag:      `hcl:"name"`,
			expected: [2]string{"name", ""},
		},
		{
			name:     "tag with modifier",
			tag:      `hcl:"type,label"`,
			expected: [2]string{"type", "label"},
		},
		{
			name:     "tag with block",
			tag:      `hcl:"config,block"`,
			expected: [2]string{"config", "block"},
		},
		{
			name:     "tag with optional",
			tag:      `hcl:"value,optional"`,
			expected: [2]string{"value", "optional"},
		},
		{
			name:     "no hcl tag",
			tag:      `json:"name"`,
			expected: [2]string{"", ""},
		},
		{
			name:     "empty tag",
			tag:      "",
			expected: [2]string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert string to StructTag
			result := parseHCLTag(reflect.StructTag(tt.tag))
			if result != tt.expected {
				t.Errorf("parseHCLTag(%q) = %v, want %v", tt.tag, result, tt.expected)
			}
		})
	}
}

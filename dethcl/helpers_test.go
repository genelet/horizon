package dethcl

import (
	"reflect"
	"testing"
)

// Test structures for parseFieldInfo and getStructFields
type TestStruct struct {
	Name        string `hcl:"name"`
	Description string `hcl:"desc,optional"`
	Label       string `hcl:"type,label"`
	Block       string `hcl:"config,block"`
	Ignored     string `hcl:"-"`
	unexported  string
	NoTag       string
}

type EmbeddedStruct struct {
	Embedded string `hcl:"embedded"`
}

type TestWithEmbedded struct {
	EmbeddedStruct
	Name string `hcl:"name"`
}

func TestParseFieldInfo(t *testing.T) {
	tests := []struct {
		name     string
		field    reflect.StructField
		value    reflect.Value
		expected *HCLFieldInfo
		isNil    bool
	}{
		{
			name: "normal field with tag",
			field: reflect.StructField{
				Name: "Name",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"name"`,
			},
			value: reflect.ValueOf("test"),
			expected: &HCLFieldInfo{
				TagName:  "name",
				Modifier: "",
				IsLabel:  false,
				IsBlock:  false,
				IsIgnore: false,
			},
		},
		{
			name: "label field",
			field: reflect.StructField{
				Name: "Type",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"type,label"`,
			},
			value: reflect.ValueOf("test"),
			expected: &HCLFieldInfo{
				TagName:  "type",
				Modifier: "label",
				IsLabel:  true,
				IsBlock:  false,
				IsIgnore: false,
			},
		},
		{
			name: "block field",
			field: reflect.StructField{
				Name: "Config",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"config,block"`,
			},
			value: reflect.ValueOf("test"),
			expected: &HCLFieldInfo{
				TagName:  "config",
				Modifier: "block",
				IsLabel:  false,
				IsBlock:  true,
				IsIgnore: false,
			},
		},
		{
			name: "ignored field with dash",
			field: reflect.StructField{
				Name: "Ignored",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"-"`,
			},
			value: reflect.ValueOf("test"),
			isNil: true,
		},
		{
			name: "unexported field",
			field: reflect.StructField{
				Name: "unexported",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"unexported"`,
			},
			value: reflect.ValueOf("test"),
			isNil: true,
		},
		{
			name: "optional field",
			field: reflect.StructField{
				Name: "Optional",
				Type: reflect.TypeOf(""),
				Tag:  `hcl:"optional,optional"`,
			},
			value: reflect.ValueOf("test"),
			expected: &HCLFieldInfo{
				TagName:  "optional",
				Modifier: "optional",
				IsLabel:  false,
				IsBlock:  false,
				IsIgnore: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseFieldInfo(tt.field, tt.value)

			if tt.isNil {
				if result != nil {
					t.Errorf("expected nil, got %+v", result)
				}
				return
			}

			if result == nil {
				t.Fatal("expected non-nil result")
			}

			if result.TagName != tt.expected.TagName {
				t.Errorf("TagName = %q, want %q", result.TagName, tt.expected.TagName)
			}
			if result.Modifier != tt.expected.Modifier {
				t.Errorf("Modifier = %q, want %q", result.Modifier, tt.expected.Modifier)
			}
			if result.IsLabel != tt.expected.IsLabel {
				t.Errorf("IsLabel = %v, want %v", result.IsLabel, tt.expected.IsLabel)
			}
			if result.IsBlock != tt.expected.IsBlock {
				t.Errorf("IsBlock = %v, want %v", result.IsBlock, tt.expected.IsBlock)
			}
		})
	}
}

func TestGetStructFields(t *testing.T) {
	t.Run("basic struct", func(t *testing.T) {
		ts := TestStruct{
			Name:        "test",
			Description: "desc",
			Label:       "label",
			Block:       "block",
			Ignored:     "ignored",
			unexported:  "unexported",
			NoTag:       "notag",
		}

		v := reflect.ValueOf(ts)
		fields, err := getStructFields(reflect.TypeOf(ts), v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have: Name, Description, Label, Block, NoTag (not Ignored or unexported)
		if len(fields) != 5 {
			t.Errorf("expected 5 fields, got %d", len(fields))
		}

		// Verify field names
		tagNames := make(map[string]bool)
		for _, f := range fields {
			tagNames[f.TagName] = true
		}

		expectedTags := []string{"name", "desc", "type", "config", "notag"}
		for _, expected := range expectedTags {
			if !tagNames[expected] {
				t.Errorf("expected tag %q not found", expected)
			}
		}
	})

	t.Run("embedded struct", func(t *testing.T) {
		ts := TestWithEmbedded{
			EmbeddedStruct: EmbeddedStruct{Embedded: "embedded"},
			Name:           "test",
		}

		v := reflect.ValueOf(ts)
		fields, err := getStructFields(reflect.TypeOf(ts), v)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have both embedded and name fields
		if len(fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(fields))
		}

		tagNames := make(map[string]bool)
		for _, f := range fields {
			tagNames[f.TagName] = true
		}

		if !tagNames["embedded"] {
			t.Error("expected embedded field not found")
		}
		if !tagNames["name"] {
			t.Error("expected name field not found")
		}
	})
}

func TestIsComplexField(t *testing.T) {
	tests := []struct {
		name     string
		value    any
		expected bool
	}{
		{
			name:     "string is simple",
			value:    "test",
			expected: false,
		},
		{
			name:     "int is simple",
			value:    42,
			expected: false,
		},
		{
			name:     "struct is complex",
			value:    struct{ Name string }{Name: "test"},
			expected: true,
		},
		{
			name:     "pointer is complex",
			value:    new(string),
			expected: true,
		},
		{
			name:     "slice of interfaces is complex",
			value:    []any{"test", 42},
			expected: true,
		},
		{
			name:     "slice of strings is simple",
			value:    []string{"a", "b"},
			expected: false,
		},
		{
			name:     "slice of structs is complex",
			value:    []struct{ Name string }{{Name: "test"}},
			expected: true,
		},
		{
			name:     "map of strings is simple",
			value:    map[string]string{"key": "value"},
			expected: false,
		},
		{
			name:     "map of structs is complex",
			value:    map[string]struct{ Name string }{"key": {Name: "test"}},
			expected: true,
		},
		{
			name:     "empty slice",
			value:    []string{},
			expected: false,
		},
		{
			name:     "empty map",
			value:    map[string]string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.value)
			result := isComplexField(v, v.Type())
			if result != tt.expected {
				t.Errorf("isComplexField() = %v, want %v", result, tt.expected)
			}
		})
	}
}

type TestCategorizeStruct struct {
	Label      string                  `hcl:"type,label"`
	Name       string                  `hcl:"name"`
	Count      int                     `hcl:"count"`
	Config     struct{ Value string }  `hcl:"config,block"`
	Tags       []string                `hcl:"tags"`
	Metadata   map[string]string       `hcl:"metadata"`
	Nested     []struct{ Name string } `hcl:"nested"`
	ZeroValue  string                  `hcl:"zero"`
	EmptySlice []string                `hcl:"empty_slice"`
	EmptyMap   map[string]string       `hcl:"empty_map"`
}

func TestCategorizeFields(t *testing.T) {
	ts := TestCategorizeStruct{
		Label:      "test",
		Name:       "myname",
		Count:      42,
		Config:     struct{ Value string }{Value: "config"},
		Tags:       []string{"a", "b"},
		Metadata:   map[string]string{"key": "value"},
		Nested:     []struct{ Name string }{{Name: "nested"}},
		ZeroValue:  "",
		EmptySlice: []string{},
		EmptyMap:   map[string]string{},
	}

	v := reflect.ValueOf(ts)
	fields, err := getStructFields(reflect.TypeOf(ts), v)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cats := categorizeFields(fields)

	// Should have 1 label
	if len(cats.Labels) != 1 {
		t.Errorf("expected 1 label field, got %d", len(cats.Labels))
	}
	if len(cats.Labels) > 0 && cats.Labels[0].TagName != "type" {
		t.Errorf("expected label 'type', got %q", cats.Labels[0].TagName)
	}

	// Should have simple fields: name, count, tags, metadata, empty_slice, empty_map (not zero string values)
	// Count should be 6 (name, count, tags, metadata, empty_slice, empty_map)
	if len(cats.Simple) != 6 {
		t.Errorf("expected 6 simple fields, got %d", len(cats.Simple))
		for _, f := range cats.Simple {
			t.Logf("  simple: %s", f.TagName)
		}
	}

	// Should have complex fields: config, nested
	if len(cats.Complex) != 2 {
		t.Errorf("expected 2 complex fields, got %d", len(cats.Complex))
		for _, f := range cats.Complex {
			t.Logf("  complex: %s", f.TagName)
		}
	}

	// Verify tag names
	simpleNames := make(map[string]bool)
	for _, f := range cats.Simple {
		simpleNames[f.TagName] = true
	}

	expectedSimple := []string{"name", "count", "tags", "metadata", "empty_slice", "empty_map"}
	for _, expected := range expectedSimple {
		if !simpleNames[expected] {
			t.Errorf("expected simple field %q not found", expected)
		}
	}

	complexNames := make(map[string]bool)
	for _, f := range cats.Complex {
		complexNames[f.TagName] = true
	}

	expectedComplex := []string{"config", "nested"}
	for _, expected := range expectedComplex {
		if !complexNames[expected] {
			t.Errorf("expected complex field %q not found", expected)
		}
	}
}
